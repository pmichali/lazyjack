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

func CreateKubeletDropInContents(c *Config) *bytes.Buffer {
	ip, _, _ := net.ParseCIDR(c.Service.CIDR) // Already validated

	contents := bytes.NewBufferString("[Service]\n")
	// Assumption is that kube-dns will be at address 10 (0xa) in service network
	fmt.Fprintf(contents, "Environment=\"KUBELET_DNS_ARGS=--cluster-dns=%sa --cluster-domain=cluster.local\"\n", ip)
	return contents
}

// Override kubelet configuration, so that the correct address is used for DNS resolution.
func CreateKubeletDropInFile(c *Config) error {
	contents := CreateKubeletDropInContents(c)

	err := os.MkdirAll(c.General.SystemdArea, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create area for kubelet drop-in file: %s", err.Error())
	}
	dropIn := filepath.Join(c.General.SystemdArea, KubeletDropInFile)
	err = ioutil.WriteFile(dropIn, contents.Bytes(), 0755)
	if err == nil {
		glog.V(1).Infof("Created kubelet drop-in file")
	}
	return err
}

func CreateKubeAdmConfigContents(n *Node, c *Config) *bytes.Buffer {
	header := `# Autogenerated file
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
kubernetesVersion: 1.9.0
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
	return contents
}

func CreateKubeAdmConfigFile(node *Node, c *Config) error {
	contents := CreateKubeAdmConfigContents(node, c)

	file := filepath.Join(c.General.WorkArea, KubeAdmConfFile)
	err := ioutil.WriteFile(file, contents.Bytes(), 0755)
	if err == nil {
		glog.V(1).Infof("Created %s file", KubeAdmConfFile)
	}
	return err
}

type NodeInfo struct {
	Name string
	IP   string
	Seen bool
}

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

func MatchingNodeIndex(line []byte, n []NodeInfo) int {
	for i, node := range n {
		if strings.Contains(string(line), node.Name) {
			return i
		}
	}
	return -1
}

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

func FindHostIPForNAT64(c *Config) (string, bool) {
	for _, node := range c.Topology {
		if node.IsNAT64Server {
			return fmt.Sprintf("%s%d", c.Mgmt.Prefix, node.ID), true
		}
	}
	return "", false
}

func CreateRouteToNAT64ServerForDNS64Subnet(node *Node, c *Config) (err error) {
	var gw string
	var ok bool
	dest := c.DNS64.CIDR
	if node.IsNAT64Server {
		gw = c.NAT64.ServerIP
		err = AddRouteUsingSupportNetInterface(dest, gw, c.Support.V4CIDR)
	} else {
		gw, ok = FindHostIPForNAT64(c)
		if !ok {
			return fmt.Errorf("Unable to find node with NAT64 server configured")
		}
		err = AddRouteUsingInterfaceName(dest, gw, node.Interface)
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

func CreateRouteToSupportNetworkForOtherNodes(node *Node, c *Config) (err error) {
	if !node.IsNAT64Server && !node.IsDNS64Server {
		dest := c.Support.CIDR
		gw, ok := FindHostIPForNAT64(c)
		if !ok {
			return fmt.Errorf("Unable to find node with NAT64 server configured")
		}
		err = AddRouteUsingInterfaceName(dest, gw, node.Interface)
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

func PrepareClusterNode(node *Node, c *Config) {
	glog.V(1).Info("Preparing general settings")

	mgmtIP := BuildNodeCIDR(c.Mgmt.Prefix, node.ID, c.Mgmt.Size)
	err := AddAddressToLink(mgmtIP, node.Interface)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	err = AddHostEntries(c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	err = AddResolvConfEntry(c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	err = CreateKubeletDropInFile(c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	if node.IsMaster {
		err = CreateKubeAdmConfigFile(node, c)
		if err != nil {
			glog.Fatalf(err.Error())
			os.Exit(1) // TODO: Rollback?
		}
	}

	err = CreateRouteToNAT64ServerForDNS64Subnet(node, c)
	if err != nil {
		glog.Fatalf(err.Error())
		os.Exit(1) // TODO: Rollback?
	}

	err = CreateRouteToSupportNetworkForOtherNodes(node, c)
	if err != nil {
		glog.Fatalf(err.Error())
		os.Exit(1) // TODO: Rollback?
	}
	glog.Info("Prepared general settings")
}

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
	trailer := `        exclude { any; };
    };
};
`
	contents := bytes.NewBufferString(header)
	fmt.Fprintf(contents, "        %s%s;\n", c.DNS64.CIDRPrefix, c.DNS64.RemoteV4Server)
	fmt.Fprintf(contents, middle)
	fmt.Fprintf(contents, "    dns64 %s {\n", c.DNS64.CIDR)
	fmt.Fprintf(contents, trailer)
	return contents
}

func CreateSupportNetwork(c *Config) {
	if ResourceExists(SupportNetName) {
		glog.V(1).Infof("Skipping - support network already exists")
		return
	}

	args := BuildCreateNetArgsForSupportNet(c.Support.CIDR, c.Support.Prefix, c.Support.V4CIDR)
	_, err := DoCommand(SupportNetName, args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.Info("Prepared support network")
	}
}

func BuildFileStructureForDNS() error {
	err := os.RemoveAll(DNS64BaseArea)
	if err != nil {
		return err
	}
	err = os.MkdirAll(DNS64ConfArea, 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(DNS64CacheArea, 0755)
	if err != nil {
		return err
	}
	return nil
}

func CreateConfigForDNS64(c *Config) error {
	err := BuildFileStructureForDNS()
	if err != nil {
		return fmt.Errorf("Unable to create directory structure for DNS64: %s", err.Error())
	}

	contents := CreateNamedConfContents(c)
	err = ioutil.WriteFile(DNS64NamedConf, contents.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("Unable to create named.conf for DNS64: %s", err.Error())
	}

	glog.V(1).Infof("Created DNS64 config file")
	return nil
}

func ParseIPv4Address(ifConfig string) string {
	re := regexp.MustCompile("(?m)^\\s+inet\\s+(\\d+[.]\\d+[.]\\d+[.]\\d+/\\d+)\\s")
	m := re.FindStringSubmatch(ifConfig)
	if len(m) == 2 {
		return m[1] // Want just the CIDR
	}
	return ""
}

// NOTE: Will use existing container, if running
func PrepareDNS64Server(node *Node, c *Config) {
	glog.V(1).Info("Preparing DNS64")

	if ResourceExists("bind9") {
		glog.V(1).Infof("Skipping - DNS64 container (bind9) already exists on %s", node.Name)
		return
	}

	err := CreateConfigForDNS64(c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	// Run DNS64 (bind9) container
	args := BuildRunArgsForDNS64(c)
	_, err = DoCommand("DNS64 container", args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(1).Info("DNS64 container (bind9) started")
	}

	// Remove IPv4 address, so only an IPv6 address in DNS64 container
	args = BuildGetInterfaceArgsForDNS64()
	ifConfig, err := DoCommand("Get I/F config", args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(4).Info("Have eth0 info for DNS64 container")
	}
	v4Addr := ParseIPv4Address(ifConfig)
	if v4Addr == "" {
		glog.Fatal("Unable to find IPv4 address on eth0 of DNS64 container")
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(4).Infof("Have IPv4 address (%s) for DNS64 container", v4Addr)
	}
	args = BuildV4AddrDelArgsForDNS64(v4Addr)
	_, err = DoCommand("Delete IPv4 addr", args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(4).Info("Deleted IPv4 address in DNS64 container")
	}

	// Create a route in container to NAT64 server, for synthesized IPv6 addresses
	args = BuildAddRouteArgsForDNS64(c)
	_, err = DoCommand("Add IPv6 route", args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(4).Info("Have IPv6 route in DNS64 container")
	}
	glog.Info("Prepared DNS64 container")
}

// NOTE: Will use existing container, if running
func PrepareNAT64Server(node *Node, c *Config) {
	glog.V(1).Info("Preparing NAT64")

	if ResourceExists("tayga") {
		glog.V(1).Infof("Skipping - NAT64 container (tayga) already exists")
		return
	}

	// Run NAT64 (tayga) container
	args := BuildRunArgsForNAT64(c)
	_, err := DoCommand("NAT64 container", args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(1).Info("NAT64 container (tayga) started")
	}

	err = AddRouteUsingSupportNetInterface(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		if err.Error() == "file exists" {
			glog.V(1).Infof("Skipping - add route to %s via %s as already exists", c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP)
		} else {
			glog.Fatal(err)
			os.Exit(1) // TODO: Rollback?
		}
	} else {
		glog.V(1).Info("Local IPv4 route added pointing to NAT64 container")
	}
	glog.Info("Prepared NAT64 server")
}

func Prepare(name string, c *Config) {
	node := c.Topology[name]
	glog.Infof("Preparing %q", name)
	// TODO: Verify docker version OK (17.03, others?), else warn...
	if node.IsDNS64Server || node.IsNAT64Server {
		// TODO: Verify that node has default IPv4 route
		CreateSupportNetwork(c)
	}
	if node.IsDNS64Server {
		PrepareDNS64Server(&node, c)
	}
	if node.IsNAT64Server {
		PrepareNAT64Server(&node, c)
	}
	if node.IsMaster || node.IsMinion {
		PrepareClusterNode(&node, c)
	}
	glog.Infof("Prepared node %q", name)
}
