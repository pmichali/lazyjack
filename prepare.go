package orca

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/golang/glog"
)

func CreateKubeletDropInContents(c *Config) *bytes.Buffer {
	contents := bytes.NewBufferString("[Service]\n")
	fmt.Fprintf(contents, "Environment=\"KUBELET_DNS_ARGS=--cluster-dns=%s --cluster-domain=cluster.local\"\n",
		c.DNS64.ServerIP)
	return contents
}

// Override kubelet configuration, so that the correct address is used for DNS resolution.
func CreateKubeletDropInFile(c *Config) error {
	contents := CreateKubeletDropInContents(c)

	err := os.MkdirAll(KubeletSystemdArea, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create kubelet drop-in file: %s", err.Error())
	}
	err = ioutil.WriteFile(KubeletDropInFile, contents.Bytes(), 0755)
	if err == nil {
		glog.V(1).Infof("Created kubelet drop-in file")
	}
	return err
}

func PrepareClusterNode(node *Node, c *Config) {
	glog.Infof("Preparing node %q", node.Name)

	mgmtIP := BuildCIDR(c.Mgmt.Subnet, node.ID, c.Mgmt.Size)
	err := AddAddressToLink(mgmtIP, node.Interface)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback
	}
	// hosts
	// resolv.conf

	err = CreateKubeletDropInFile(c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback
	}
	glog.Infof("Prepared node %s", node.Name)
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
	fmt.Fprintf(contents, "        %s%s;\n", c.DNS64.Prefix, c.DNS64.RemoteV4Server)
	fmt.Fprintf(contents, middle)
	fmt.Fprintf(contents, "    dns64 %s/%d {\n", c.DNS64.Prefix, c.DNS64.PrefixSize)
	fmt.Fprintf(contents, trailer)
	return contents
}

func CreateSupportNetwork(c *Config) {
	if ResourceExists(SupportNetName) {
		glog.V(1).Infof("Skipping - support network already exists")
		return
	}

	args := BuildCreateNetArgsForSupportNet(c.Support.Subnet, c.Support.Size, c.Support.V4CIDR)
	_, err := DoCommand(SupportNetName, args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.Info("Created support network")
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
	glog.Infof("Preparing DNS64 on %q", node.Name)

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
	glog.Info("DNS64 container configured on %s", node.Name)
}

// NOTE: Will use existing container, if running
func PrepareNAT64Server(node *Node, c *Config) {
	glog.V(1).Infof("Preparing NAT64 on %q", node.Name)

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

	err = AddLocalV4RouteToNAT64Server(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(1).Info("Local IPv4 route added pointing to NAT64 container")
	}
	glog.Infof("Prepared NAT64 server on %q", node.Name)
}

func EnsureCNIAreaExists() error {
	err := os.RemoveAll(CNIConfArea)
	if err != nil {
		return err
	}
	err = os.MkdirAll(CNIConfArea, 0755)
	if err != nil {
		return err
	}
	return nil
}

func PreparePlugin(node *Node, c *Config) {
	glog.V(1).Infof("Preparing %s plugin on %q", c.Plugin, node.Name)
	err := EnsureCNIAreaExists()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(4).Info("Created area for CNI config file")
	}
	err = CreateBridgeCNIConfigFile(node, c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}
	glog.Info("Prepared for %s plugin", c.Plugin)
}

func Prepare(name string, c *Config) {
	node := c.Topology[name]
	glog.V(4).Infof("Preparing %q -> %+v", name, node)
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
		PreparePlugin(&node, c)
	}
}
