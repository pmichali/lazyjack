package orca

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

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
		glog.V(1).Info("Created support network")
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

// NOTE: Will use existing container, if running
func PrepareDNS64Server(node *Node, c *Config) {
	glog.Infof("Preparing DNS64 on %q", node.Name)

	if ResourceExists("bind9") {
		glog.V(1).Infof("Skipping - DNS64 container (bind9) already exists")
		return
	}

	err := CreateConfigForDNS64(c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	args := BuildRunArgsForDNS64(c)
	_, err = DoCommand("bind9", args)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(1).Info("DNS64 container (bind9) started")
	}

	//    Pull IPv4 address
	//    Add V6 route
	// Ensure default V4 route
}

func PrepareNAT64Server(node *Node, c *Config) {
	glog.Infof("Preparing NAT64 on %q", node.Name)
	// See if already exists
	// Create container
	// Add route to V4 subnet in container
	// Add V6 route to NAT server
	// Ensure default V4 route
}

func PreparePlugin(node *Node, c *Config) {
	glog.Infof("Preparing bridge plugin on %q", node.Name)
	// For bridge plugin create CNI config file
}

func Prepare(name string, c *Config) {
	node := c.Topology[name]
	glog.V(4).Infof("Preparing %q -> %+v", name, node)
	if node.IsMaster || node.IsMinion {
		PrepareClusterNode(&node, c)
		PreparePlugin(&node, c)
	}
	if node.IsDNS64Server || node.IsNAT64Server {
		CreateSupportNetwork(c)
	}
	if node.IsDNS64Server {
		PrepareDNS64Server(&node, c)
	}
	if node.IsNAT64Server {
		PrepareNAT64Server(&node, c)
	}
}
