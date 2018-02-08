package orca

import (
	"os"

	"github.com/golang/glog"
)

func CleanupClusterNode(node *Node, c *Config) {
	glog.Infof("Cleaning cluster node %q", node.Name)
	err := os.Remove(KubeletDropInFile)
	if err != nil {
		glog.Warningf("Unable to remove kubelet drop-in file (%s): %s", KubeletDropInFile, err.Error())
	}
	glog.V(1).Info("Removed kubelet drop-in file")

	mgmtIP := BuildCIDR(c.Mgmt.Subnet, node.ID, c.Mgmt.Size)
	err = RemoveAddressFromLink(mgmtIP, node.Interface)
	if err != nil {
		glog.Warning("Unable to remove IP from management interface: %s", err.Error())
	}

	// Will leave /etc/hosts and /etc/resolv.conf alone
}

func CleanupDNS64Server(node *Node, c *Config) {
	glog.Infof("Cleaning DNS64 on %q", node.Name)
	// Remove Container
	// Will leave V4 default route
}

func CleanupNAT64Server(node *Node, c *Config) {
	glog.Infof("Cleaning NAT64 on %q", node.Name)
	// Delete route to V4 subnet in container
	// Delete V6 route to NAT server
	// Delete container
	// Will leave default V4 route
}

func CleanupPlugin(node *Node, c *Config) {
	glog.Infof("Cleaning bridge plugin on %q", node.Name)
	// For bridge plugin remove CNI config file (should be deleted when "kubeadm reset")
}

func Cleanup(name string, c *Config) {
	node := c.Topology[name]
	glog.V(4).Infof("Cleaning %q -> %+v", name, node)
	if node.IsMaster || node.IsMinion {
		CleanupClusterNode(&node, c)
		CleanupPlugin(&node, c)
	}
	if node.IsDNS64Server {
		CleanupDNS64Server(&node, c)
	}
	if node.IsNAT64Server {
		CleanupNAT64Server(&node, c)
	}
}
