package lazyjack

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/golang/glog"
)

// CreateKubeletDropInContents constructs the contents of the kubelet
// drop-in file to support IPv6.
func CreateKubeletDropInContents(c *Config) *bytes.Buffer {
	ip, _, _ := net.ParseCIDR(c.Service.CIDR) // Already validated

	contents := bytes.NewBufferString("[Service]\n")
	// Assumption is that kube-dns will be at address 10 (0xa) in service network
	fmt.Fprintf(contents, "Environment=\"KUBELET_DNS_ARGS=--cluster-dns=%sa --cluster-domain=cluster.local\"\n", ip)
	return contents
}

// CreateKubeletDropInFile creates a config file to override the kubelet
// configuration, so that the correct address is used for DNS resolution.
func CreateKubeletDropInFile(c *Config) error {
	contents := CreateKubeletDropInContents(c)

	err := os.MkdirAll(c.General.SystemdArea, 0755)
	if err != nil {
		return fmt.Errorf("unable to create area for kubelet drop-in file: %v", err)
	}
	dropIn := filepath.Join(c.General.SystemdArea, KubeletDropInFile)
	err = ioutil.WriteFile(dropIn, contents.Bytes(), 0755)
	if err == nil {
		glog.V(1).Infof("Created kubelet drop-in file")
	}
	return err
}

// CreateKubeAdmConfigContents constructs the contents of the KubeAdm
// configuration file for IPv6 operation.
func CreateKubeAdmConfigContents(n *Node, c *Config) []byte {
	header := `# Autogenerated file
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
api:
`
	trailer := `tokenTTL: 0s
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
  runtime-config: "admissionregistration.k8s.io/v1alpha1"
  feature-gates: AllAlpha=true
`
	contents := bytes.NewBufferString(header)
	fmt.Fprintf(contents, "  advertiseAddress: \"%s%d\"\n", c.Mgmt.Prefix, n.ID)
	fmt.Fprintf(contents, "networking:\n")
	fmt.Fprintf(contents, "  serviceSubnet: %q\n", c.Service.CIDR)
	fmt.Fprintf(contents, "nodeName: %s\n", n.Name)
	fmt.Fprintf(contents, "token: %q\n", c.General.Token)
	fmt.Fprintf(contents, trailer)
	return contents.Bytes()
}

// CreateKubeAdmConfigFile constructs the KubeAdm config file during the
// "prepare" step. This file can be modified, before using it in the "up"
// step.
func CreateKubeAdmConfigFile(node *Node, c *Config) error {
	contents := CreateKubeAdmConfigContents(node, c)

	file := filepath.Join(c.General.WorkArea, KubeAdmConfFile)
	backup := fmt.Sprintf("%s.bak", file)
	err := SaveFileContents(contents, file, backup)
	if err == nil {
		glog.V(1).Infof("Created %s file", KubeAdmConfFile)
	}
	return err
}

// NodeInfo holds name, IP address, and an indication that the node
// has been "visited".
type NodeInfo struct {
	Name string
	IP   string
	Seen bool
}

// BuildNodeInfo creates a slice with all information on all the nodes
// sorted in alphabetical order.
func BuildNodeInfo(c *Config) []NodeInfo {
	n := make([]NodeInfo, len(c.Topology))
	i := 0
	for nodeName, node := range c.Topology {
		ip := fmt.Sprintf("%s%d", c.Mgmt.Prefix, node.ID)
		glog.V(4).Infof("Created node info for %s (%s)", nodeName, ip)
		n[i] = NodeInfo{Name: nodeName, IP: ip, Seen: false}
		i++
	}
	// Since it is a map of nodes, sort, so output is predictable
	sort.Slice(n, func(i, j int) bool {
		return n[i].Name < n[j].Name
	})
	return n
}

// MatchingNodeIndex obtains the index of the node entry that matches
// the name of one of the existing nodes.
func MatchingNodeIndex(line []byte, n []NodeInfo) int {
	for i, node := range n {
		if strings.Contains(string(line), node.Name) {
			return i
		}
	}
	return -1
}

// UpdateHostsInfo goes through the /etc/hosts file and updates
// the IP addresses for nodes that are called out in the configuration
// file. Any existing entry is commented out, with a special tag to allow
// restoration. New entries get a comment that can be use to remove them
// upon cleanup.
func UpdateHostsInfo(contents []byte, n []NodeInfo) []byte {
	glog.V(4).Infof("Updating %s", EtcHostsFile)
	lines := bytes.Split(bytes.TrimRight(contents, "\n"), []byte("\n"))
	var output bytes.Buffer
	for _, line := range lines {
		if bytes.HasSuffix(line, []byte("  #[+]")) {
			continue // prepare was previousy run, filter out additions
		}
		if !bytes.HasPrefix(line, []byte("#")) {
			i := MatchingNodeIndex(line, n)
			if i >= 0 {
				if strings.Contains(string(line), n[i].IP) {
					n[i].Seen = true
				} else {
					output.WriteString("#[-] ")
				}
			}
		}
		output.WriteString(fmt.Sprintf("%s\n", line))
	}
	// Create any missing entries
	for _, node := range n {
		if !node.Seen {
			output.WriteString(fmt.Sprintf("%s %s  #[+]\n", node.IP, node.Name))
		}
	}
	return output.Bytes()
}

// AddHostEntries udpates the /etc/hosts file with IP addresses
// to be used by the cluster for each node.
func AddHostEntries(c *Config) error {
	file := filepath.Join(c.General.EtcArea, EtcHostsFile)
	backup := filepath.Join(c.General.EtcArea, EtcHostsBackupFile)

	glog.V(1).Infof("Preparing %s file", file)
	nodes := BuildNodeInfo(c)
	contents, err := GetFileContents(file)
	if err != nil {
		return err
	}
	contents = UpdateHostsInfo(contents, nodes)
	err = SaveFileContents(contents, file, backup)
	if err != nil {
		return err
	}
	glog.Infof("Prepared %s file", file)
	return nil
}

// UpdateResolvConfInfo updates the nameservers to use the ones
// defined for the cluster. Old entries are commented out, and new
// ones tagged, allowing later restoration, during cleanup.
func UpdateResolvConfInfo(contents []byte, ns string) []byte {
	glog.V(4).Infof("Updating %s", EtcResolvConfFile)
	lines := bytes.Split(bytes.TrimRight(contents, "\n"), []byte("\n"))

	var output bytes.Buffer
	first := true
	for _, line := range lines {
		if bytes.HasSuffix(line, []byte("  #[+]")) {
			continue // prepare was previousy run, filter out additions
		}
		if bytes.HasPrefix(line, []byte("nameserver")) {
			matches := bytes.Contains(line, []byte(ns))
			if first && !matches {
				output.WriteString(fmt.Sprintf("nameserver %s  #[+]\n", ns))
			} else if !first && matches {
				output.WriteString("#[-] ")
			} // else first and matches, or not first an not matches -> keep line
			if first {
				first = false
			}
		}
		output.WriteString(fmt.Sprintf("%s\n", line))
	}
	if first {
		output.WriteString(fmt.Sprintf("nameserver %s  #[+]\n", ns))
	}
	return output.Bytes()
}

// AddResolvConfEntry updates the /etc/resolv.conf file with nameserver
// entry used for the cluster.
func AddResolvConfEntry(c *Config) error {
	file := filepath.Join(c.General.EtcArea, EtcResolvConfFile)
	backup := filepath.Join(c.General.EtcArea, EtcResolvConfBackupFile)
	glog.V(1).Infof("Preparing %s file", file)
	contents, err := GetFileContents(file)
	if err != nil {
		return err
	}
	contents = UpdateResolvConfInfo(contents, c.DNS64.ServerIP)
	err = SaveFileContents(contents, file, backup)
	if err != nil {
		return err
	}
	glog.Infof("Prepared %s file", file)
	return nil
}

// FindHostIPForNAT64 determines the management IP for the node containing
// the NAT64 server.
func FindHostIPForNAT64(c *Config) (string, bool) {
	for _, node := range c.Topology {
		if node.IsNAT64Server {
			return fmt.Sprintf("%s%x", c.Mgmt.Prefix, node.ID), true
		}
	}
	return "", false
}

// CreateRouteToNAT64ServerForDNS64Subnet creates a route for the DNS64
// network that points to the NAT64 server for proper routing of external
// addresses.
func CreateRouteToNAT64ServerForDNS64Subnet(node *Node, c *Config) (err error) {
	var gw string
	var ok bool
	dest := c.DNS64.CIDR
	if node.IsNAT64Server {
		gw = c.NAT64.ServerIP
		err = c.General.NetMgr.AddRouteUsingSupportNetInterface(dest, gw, c.Support.V4CIDR)
	} else {
		gw, ok = FindHostIPForNAT64(c)
		if !ok {
			return fmt.Errorf("unable to find node with NAT64 server configured")
		}
		err = c.General.NetMgr.AddRouteUsingInterfaceName(dest, gw, node.Interface)
	}
	if err != nil {
		if err.Error() == "file exists" {
			glog.V(1).Infof("Skipping - add route to %s via %s as already exists", dest, gw)
			err = nil
		}
	} else {
		glog.V(1).Infof("Added route to %s via %s", dest, gw)
	}
	return err
}

// CreateRouteToSupportNetworkForOtherNodes creates a route on a node, to
// get to the support netork, so that the DNS64 and NAT64 server can be
// accessed.
func CreateRouteToSupportNetworkForOtherNodes(node *Node, c *Config) (err error) {
	if !node.IsNAT64Server && !node.IsDNS64Server {
		dest := c.Support.CIDR
		gw, ok := FindHostIPForNAT64(c)
		if !ok {
			return fmt.Errorf("unable to find node with NAT64 server configured")
		}
		err = c.General.NetMgr.AddRouteUsingInterfaceName(dest, gw, node.Interface)
		if err != nil {
			if err.Error() == "file exists" {
				glog.V(1).Infof("Skipping - add route to %s via %s as already exists", dest, gw)
				err = nil
			}
		} else {
			glog.V(1).Infof("Added route to %s via %s", dest, gw)
		}
	}
	return err
}

// ConfigureManagementInterface adds and address and sets the MTU for
// the interface used for the pod and management networks.
func ConfigureManagementInterface(node *Node, c *Config) error {
	glog.V(1).Infof("Configuring management interface %s", node.Interface)
	mgmtIP := BuildNodeCIDR(c.Mgmt.Prefix, node.ID, c.Mgmt.Size)
	err := c.General.NetMgr.AddAddressToLink(mgmtIP, node.Interface)
	if err != nil {
		return err
	}
	err = c.General.NetMgr.SetLinkMTU(node.Interface, c.Pod.MTU)
	if err == nil {
		glog.V(4).Infof("Set %s IP to %s and MTU to %d", node.Interface, mgmtIP, c.Pod.MTU)
	}
	return err
}

// PrepareClusterNode performs steps on the node to prepare for bringing
// up the cluster. Includes adding the management IP, updating hosts and
// resolv.conf entries, creating a kubelet drop-in file, creating the
// KubeAdm configuration file (on master), and creating routes to servers
// and the support network.
func PrepareClusterNode(node *Node, c *Config) error {
	glog.V(1).Info("Preparing general settings")

	var err error

	err = ConfigureManagementInterface(node, c)
	if err != nil {
		return err
	}

	err = AddHostEntries(c)
	if err != nil {
		return err
	}

	err = AddResolvConfEntry(c)
	if err != nil {
		return err
	}

	err = CreateKubeletDropInFile(c)
	if err != nil {
		return err
	}

	if node.IsMaster {
		err = CreateKubeAdmConfigFile(node, c)
		if err != nil {
			return err
		}
	}

	err = CreateRouteToNAT64ServerForDNS64Subnet(node, c)
	if err != nil {
		return err
	}

	err = CreateRouteToSupportNetworkForOtherNodes(node, c)
	if err != nil {
		return err
	}
	glog.Info("Prepared general settings")
	return nil
}

// CreateNamedConfContents builds the contents of the configuration
// file used by the DNS64 server.
func CreateNamedConfContents(c *Config) *bytes.Buffer {
	header := `options {
    directory "/var/bind";
    allow-query { any; };
    forwarders {
`
	middle := `    };
    auth-nxdomain no;    # conform to RFC1035
    listen-on-v6 { any; };
`
	trailer := `    };
};
`
	contents := bytes.NewBufferString(header)
	fmt.Fprintf(contents, "        %s%s;\n", c.DNS64.CIDRPrefix, c.DNS64.RemoteV4Server)
	fmt.Fprintf(contents, middle)
	fmt.Fprintf(contents, "    dns64 %s {\n", c.DNS64.CIDR)
	if !c.DNS64.AllowAAAAUse {
		fmt.Fprintf(contents, "        exclude { any; };\n")
	}
	fmt.Fprintf(contents, trailer)
	return contents
}

// CreateSupportNetwork creates the network used by the DNS64 and NAT64
// servers.
func CreateSupportNetwork(c *Config) (err error) {
	if c.General.Hyper.ResourceState(SupportNetName) == ResourceExists {
		err = fmt.Errorf("skipping - support network already exists")
		glog.V(1).Infof(err.Error())
		return err
	}

	err = c.General.Hyper.CreateNetwork(SupportNetName, c.Support.CIDR, c.Support.V4CIDR, c.Support.Prefix)
	if err != nil {
		return err
	}
	glog.Info("Prepared support network")
	return nil
}

// BuildFileStructureForDNS creates the directory tree that will be
// used in the DNS64 server container.
func BuildFileStructureForDNS(base string) error {
	d := filepath.Join(base, DNS64BaseArea)
	err := os.RemoveAll(d)
	if err != nil {
		return err
	}
	conf := filepath.Join(d, DNS64ConfArea)
	err = os.MkdirAll(conf, 0755)
	if err != nil {
		return err
	}
	cache := filepath.Join(d, DNS64CacheArea)
	err = os.MkdirAll(cache, 0755)
	if err != nil {
		return err
	}
	return nil
}

// CreateConfigForDNS64 creates the needed configuration files for the
// DNS64 server.
func CreateConfigForDNS64(c *Config) error {
	err := BuildFileStructureForDNS(c.General.WorkArea)
	if err != nil {
		return fmt.Errorf("unable to create directory structure for DNS64: %v", err)
	}

	contents := CreateNamedConfContents(c)
	conf := filepath.Join(c.General.WorkArea, DNS64BaseArea, DNS64ConfArea, DNS64NamedConf)
	err = ioutil.WriteFile(conf, contents.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("unable to create named.conf for DNS64: %v", err)
	}

	glog.V(1).Infof("Created DNS64 config file")
	return nil
}

// ParseIPv4Address extracts the CIDR from the interface's list of IP
// addresses.
func ParseIPv4Address(ifConfig string) string {
	re := regexp.MustCompile("(?m)^\\s+inet\\s+(\\d+[.]\\d+[.]\\d+[.]\\d+/\\d+)\\s")
	m := re.FindStringSubmatch(ifConfig)
	if len(m) == 2 {
		return m[1] // Want just the CIDR
	}
	return ""
}

// EnsureDNS64Server runs the DNS64 server, if it is not running. If it
// exists, but is not running, it is first deleted. If it is running, no
// action is taken.
func EnsureDNS64Server(c *Config) (err error) {
	glog.V(1).Info("Preparing DNS64")

	state := c.General.Hyper.ResourceState(DNS64Name)
	if state == ResourceRunning {
		err = fmt.Errorf("skipping - DNS64 container (%s) already running", DNS64Name)
		glog.V(1).Info(err.Error())
		return err
	}
	if state == ResourceExists {
		err = c.General.Hyper.DeleteContainer(DNS64Name)
		if err != nil {
			return fmt.Errorf("unable to remove existing (non-running) DNS64 container: %v", err)
		}
	}

	err = CreateConfigForDNS64(c)
	if err != nil {
		return err
	}

	// Run DNS64 (bind9) container
	args := BuildRunArgsForDNS64(c)
	err = c.General.Hyper.RunContainer("DNS64 container", args)
	if err == nil {
		glog.V(1).Info("DNS64 container (%s) started", DNS64Name)
	}
	return err
}

// RemoveIPv4AddressOnDNS64Server removes IPv4 address in container,
// so there is only an IPv6 address.
func RemoveIPv4AddressOnDNS64Server(c *Config) (err error) {
	ifConfig, err := c.General.Hyper.GetInterfaceConfig(DNS64Name, "eth0")
	if err != nil {
		return err
	}
	glog.V(4).Info("Have eth0 info for DNS64 container")

	v4Addr := ParseIPv4Address(ifConfig)
	if v4Addr == "" {
		return fmt.Errorf("unable to find IPv4 address on eth0 of DNS64 container")
	}
	glog.V(4).Infof("Have IPv4 address (%s) for DNS64 container", v4Addr)

	err = c.General.Hyper.DeleteV4Address(DNS64Name, v4Addr)
	if err == nil {
		glog.V(4).Info("Deleted IPv4 address in DNS64 container")
	}
	return err
}

// AddRouteForDNS64Network creates a route in the container to the NAT64
// server for synthesized IPv6 addresses.
func AddRouteForDNS64Network(c *Config) error {
	err := c.General.Hyper.AddV6Route(DNS64Name, c.DNS64.CIDR, c.NAT64.ServerIP)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 2") {
			err = fmt.Errorf("skipping - add route to %s via %s as already exists", c.DNS64.CIDR, c.NAT64.ServerIP)
			glog.V(1).Infof(err.Error())
		}
		return err
	}
	glog.V(4).Info("Have IPv6 route in DNS64 container")
	return err
}

// PrepareDNS64Server starts up the bind9 DNS64 server. Will use
// existing container, if running. Will remove IPv4 address in the
// container and add a route to the container.
func PrepareDNS64Server(c *Config) error {
	err := EnsureDNS64Server(c)
	if err != nil && !strings.HasPrefix(err.Error(), "skipping") {
		return err
	}

	err = RemoveIPv4AddressOnDNS64Server(c)
	if err != nil && !strings.HasPrefix(err.Error(), "unable to find IPv4 address") {
		return err
	}

	err = AddRouteForDNS64Network(c)
	if err != nil && !strings.HasPrefix(err.Error(), "skipping") {
		return err
	}
	glog.Info("Prepared DNS64 container")
	return nil
}

// EnsureNAT64Server creates the NAT64 container. If it is already
// running, no action is taken. If it exists, but is not running, it
// is deleted first.
func EnsureNAT64Server(c *Config) (err error) {
	glog.V(1).Info("Preparing NAT64")
	state := c.General.Hyper.ResourceState(NAT64Name)
	if state == ResourceRunning {
		err = fmt.Errorf("skipping - NAT64 container (%s) already running", NAT64Name)
		glog.V(1).Info(err.Error())
		return err
	}
	if state == ResourceExists {
		err = c.General.Hyper.DeleteContainer(NAT64Name)
		if err != nil {
			return fmt.Errorf("unable to remove existing (non-running) NAT64 container: %v", err)
		}
	}

	// Run NAT64 (tayga) container
	args := BuildRunArgsForNAT64(c)
	err = c.General.Hyper.RunContainer("NAT64 container", args)
	if err != nil {
		return err
	}
	glog.V(1).Infof("NAT64 container (%s) started", NAT64Name)
	return nil
}

// EnsureRouteToNAT64 adds a route to the NAT64 container via the
// support network.
func EnsureRouteToNAT64(c *Config) error {
	err := c.General.NetMgr.AddRouteUsingSupportNetInterface(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		if err.Error() == "file exists" {
			err = fmt.Errorf("skipping - add route to %s via %s as already exists", c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP)
			glog.V(1).Infof(err.Error())
		}
		return err
	}
	glog.V(1).Info("Local IPv4 route added pointing to NAT64 container")
	return nil
}

// PrepareNAT64Server starts up the Tayga NAT64 server.
// NOTE: Will use existing container, if running
func PrepareNAT64Server(c *Config) error {
	err := EnsureNAT64Server(c)
	if err != nil && !strings.HasPrefix(err.Error(), "skipping") {
		return err
	}

	err = EnsureRouteToNAT64(c)
	if err != nil && !strings.HasPrefix(err.Error(), "skipping") {
		return err
	}
	glog.Info("Prepared NAT64 container")
	return nil
}

// Prepare gets ready to start up the cluster. The support network
// is created (if not on the NAT64/DNS64 node), the NAT64 and DNS64
// servers are started, and the node is configured for running the
// cluster.
func Prepare(name string, c *Config) error {
	node := c.Topology[name]
	glog.Infof("Preparing %q", name)
	var err error

	// TODO: Verify docker version OK (17.03, others?), else warn...

	if node.IsDNS64Server || node.IsNAT64Server {
		// TODO: Verify that node has default IPv4 route
		err = CreateSupportNetwork(c)
		if err != nil && !strings.HasPrefix(err.Error(), "skipping") {
			return err
		}
	}
	if node.IsDNS64Server {
		err = PrepareDNS64Server(c)
		if err != nil {
			return err
		}
	}
	if node.IsNAT64Server {
		err = PrepareNAT64Server(c)
		if err != nil {
			return err
		}
	}
	if node.IsMaster || node.IsMinion {
		err = PrepareClusterNode(&node, c)
		if err != nil {
			return err
		}
	}
	glog.Infof("Prepared node %q", name)
	return nil
}
