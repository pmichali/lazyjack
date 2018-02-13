package orca

import (
	"fmt"
	"os"

	"github.com/golang/glog"
)

func RestoreFile(backup, file string) error {
	glog.V(4).Infof("Restoring %s", file)
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		return err
	}
	err := os.Rename(backup, file)
	if err != nil {
		return err
	}
	return nil
}

func CleanupClusterNode(node *Node, c *Config) {
	glog.V(1).Info("Cleaning general settings")
	err := os.Remove(KubeletDropInFile)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(4).Info("No kubelet drop-in file to remove")
		} else {
			glog.Warningf("Unable to remove kubelet drop-in file (%s): %s", KubeletDropInFile, err.Error())
		}
	} else {
		glog.V(4).Info("Removed kubelet drop-in file")
	}

	mgmtIP := BuildNodeCIDR(c.Mgmt.Subnet, node.ID, c.Mgmt.Size)
	err = RemoveAddressFromLink(mgmtIP, node.Interface)
	if err != nil {
		glog.Warningf("Unable to remove IP from management interface: %s", err.Error())
	} else {
		glog.V(4).Info("Removed IP address from management interface")
	}

	err = RestoreFile(EtcHostsBackupFile, EtcHostsFile)
	if err != nil {
		glog.Warningf("Unable to restore %s: %s", EtcHostsFile, err.Error())
	} else {
		glog.V(4).Infof("Restored %s contents", EtcHostsFile)
	}

	err = RestoreFile(EtcResolvConfBackupFile, EtcResolvConfFile)
	if err != nil {
		glog.Warningf("Unable to restore %s: %s", EtcResolvConfFile, err.Error())
	} else {
		glog.V(4).Infof("Restored %s contents", EtcResolvConfFile)
	}

	dest := fmt.Sprintf("%s/%d", c.DNS64.Prefix, c.DNS64.PrefixSize)
	var gw string
	if node.IsNAT64Server {
		gw = c.NAT64.ServerIP
		err = DeleteRouteUsingSupportNetInterface(dest, gw, c.Support.V4CIDR)
	} else {
		gw, ok := FindHostIPForNAT64(c)
		if !ok {
			err = fmt.Errorf("Unable to find node with NAT64 server")
		} else {
			err = DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
		}
	}
	if err != nil {
		glog.Warningf("Unable to delete route to %s via %s: %s", dest, gw, err.Error())
	} else {
		glog.V(4).Infof("Deleted route to %s via %s", dest, gw)
	}

	glog.Info("Cleaned general settings")
}

func CleanupDNS64Server(node *Node, c *Config) {
	glog.V(1).Info("Cleaning DNS64")
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
	glog.Info("Cleaned DNS64")
}

func CleanupNAT64Server(node *Node, c *Config) {
	glog.V(1).Info("Cleaning NAT64")

	err := DeleteRouteUsingSupportNetInterface(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		glog.Warning(err)
	} else {
		glog.V(1).Info("Removed local IPv4 route to NAT64 container")
	}

	err = RemoveNAT64Container()
	if err != nil {
		glog.Warning("Unable to remove NAT64 container")
	} else {
		glog.V(1).Info("Removed NAT64 container")
	}
	// Will leave default V4 route
	glog.Info("Cleaned NAT64")
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
	glog.Info("Cleaned support network")
}

func CleanupPlugin(node *Node, c *Config) {
	glog.V(1).Info("Cleaning plugin")

	//	err := RemoveRoutesForPodNetwork(node, c)
	//	if err != nil {
	//		glog.Warningf("Unable to remove routes for %s plugin: %s", c.Plugin, err.Error())
	//	} else {
	//		glog.V(1).Infof("Removed routes for %s plugin", c.Plugin)
	//	}

	// Note: CNI config file will be removed, when "kubeadm reset" performed
	err := os.RemoveAll(CNIConfArea)
	if err != nil {
		glog.Warningf("Unable to remove CNI config file and area: %s", err.Error())
	} else {
		glog.V(1).Info("Removed CNI config file and area")
	}
	glog.Info("Cleaned plugin")
}

func Cleanup(name string, c *Config) {
	node := c.Topology[name]
	glog.Infof("Cleaning %q", name)
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
