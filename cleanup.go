package orca

import (
	"os"

	"github.com/golang/glog"
)

func CleanupClusterNode(node *Node, c *Config) {
	glog.Infof("Cleaning cluster node %q", node.Name)
	err := os.Remove(KubeletDropInFile)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(1).Info("No kubelet drop-in file to remove")
		} else {
			glog.Warningf("Unable to remove kubelet drop-in file (%s): %s", KubeletDropInFile, err.Error())
		}
	} else {
		glog.V(1).Info("Removed kubelet drop-in file")
	}

	mgmtIP := BuildNodeCIDR(c.Mgmt.Subnet, node.ID, c.Mgmt.Size)
	err = RemoveAddressFromLink(mgmtIP, node.Interface)
	if err != nil {
		glog.Warningf("Unable to remove IP from management interface: %s", err.Error())
	} else {
		glog.V(1).Info("Removed IP address from management interface")
	}

	// Will leave /etc/hosts and /etc/resolv.conf alone
}

func CleanupDNS64Server(node *Node, c *Config) {
	glog.Infof("Cleaning DNS64 on %q", node.Name)
	err := RemoveDNS64Container()
	if err != nil {
		glog.Warning("Unable to remove DNS64 container")
	} else {
		glog.V(1).Info("Removed DNS64 container")
	}

	err = os.RemoveAll(DNS64BaseArea)
	if err != nil {
		glog.Warning("Unable to remove DNS64 file structure")
	} else {
		glog.V(1).Info("Removed DNS64 file structure")
	}
	// Will leave V4 default route
}

func CleanupNAT64Server(node *Node, c *Config) {
	glog.Infof("Cleaning NAT64 on %q", node.Name)

	err := RemoveLocalRouteFromNAT64(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		glog.Warning(err)
	} else {
		glog.V(1).Info("Removed IPv4 route to NAT64 container")
	}

	err = RemoveNAT64Container()
	if err != nil {
		glog.Warning("Unable to remove NAT64 container")
	} else {
		glog.V(1).Info("Removed NAT64 container")
	}
	// Will leave default V4 route
}

func CleanupSupportNetwork() {
	if !ResourceExists(SupportNetName) {
		glog.V(1).Infof("Skipping - support network does not exists")
		return
	}

	args := BuildDeleteNetArgsForSupportNet()
	_, err := DoCommand("SupportNetName", args)
	if err != nil {
		glog.Warning("Unable to remove support network")
	} else {
		glog.V(1).Info("Removed support network")
	}
}

func CleanupPlugin(node *Node, c *Config) {
	glog.V(1).Infof("Cleaning plugin on %s", node.Name)

	err := RemoveRoutesForPodNetwork(node, c)
	if err != nil {
		glog.Warningf("Unable to remove routes for %s plugin: %s", c.Plugin, err.Error())
	} else {
		glog.V(1).Infof("Removed routes for %s plugin", c.Plugin)
	}

	// Note: CNI config file will be removed, when "kubeadm reset" performed
	err = os.RemoveAll(CNIConfArea)
	if err != nil {
		glog.Warningf("Unable to remove CNI config file and area: %s", err.Error())
	} else {
		glog.V(1).Info("Removed CNI config file and area")
	}
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
	if node.IsDNS64Server || node.IsNAT64Server {
		CleanupSupportNetwork()
	}
}
