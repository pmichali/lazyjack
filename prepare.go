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

func PrepareDNS64Server(node *Node, c *Config) {
	glog.Infof("Preparing DNS64 on %q", node.Name)
	// Create config file
	// Start container
	//    Pull IPv4 address
	//    Add V6 route
	// Ensure default V4 route
}

func PrepareNAT64Server(node *Node, c *Config) {
	glog.Infof("Preparing NAT64 on %q", node.Name)
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
	if node.IsDNS64Server {
		PrepareDNS64Server(&node, c)
	}
	if node.IsNAT64Server {
		PrepareNAT64Server(&node, c)
	}
}
