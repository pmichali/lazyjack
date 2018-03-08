package lazyjack

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
)

func RevertConfigInfo(contents []byte, file string) []byte {
	glog.V(4).Infof("Reverting %s", file)
	lines := bytes.Split(bytes.TrimRight(contents, "\n"), []byte("\n"))
	var output bytes.Buffer
	for _, line := range lines {
		line = bytes.TrimPrefix(line, []byte("#[-] "))
		if !bytes.HasSuffix(line, []byte("  #[+]")) {
			output.WriteString(fmt.Sprintf("%s\n", line))
		}
	}
	return output.Bytes()
}

func RevertEntries(file, backup string) error {
	glog.V(4).Infof("Cleaning %s file", file)
	contents, err := GetFileContents(file)
	if err != nil {
		glog.Warningf("Unable to read file %s to revert: %s", file, err.Error())
		return err
	}
	contents = RevertConfigInfo(contents, file)
	err = SaveFileContents(contents, file, backup)
	if err != nil {
		glog.Warningf("Unable to revert %s: %s", file, err.Error())
		return err
	}
	glog.V(4).Infof("Restored %s contents", file)
	return nil
}

func CleanupClusterNode(node *Node, c *Config) {
	glog.V(1).Info("Cleaning general settings")
	file := filepath.Join(c.General.SystemdArea, KubeletDropInFile)
	err := os.Remove(file)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(4).Info("No kubelet drop-in file to remove")
		} else {
			glog.Warningf("Unable to remove kubelet drop-in file (%s): %s", file, err.Error())
		}
	} else {
		glog.V(4).Info("Removed kubelet drop-in file")
	}

	mgmtIP := BuildNodeCIDR(c.Mgmt.Prefix, node.ID, c.Mgmt.Size)
	err = RemoveAddressFromLink(mgmtIP, node.Interface)
	if err != nil {
		glog.Warningf("Unable to remove IP from management interface: %s", err.Error())
	} else {
		glog.V(4).Info("Removed IP address from management interface")
	}

	file = filepath.Join(c.General.EtcArea, EtcHostsFile)
	backup := filepath.Join(c.General.EtcArea, EtcHostsBackupFile)
	RevertEntries(file, backup)

	file = filepath.Join(c.General.EtcArea, EtcResolvConfFile)
	backup = filepath.Join(c.General.EtcArea, EtcResolvConfBackupFile)
	RevertEntries(file, backup)

	dest := c.DNS64.CIDR
	var gw string
	var ok bool
	if node.IsNAT64Server {
		gw = c.NAT64.ServerIP
		err = DeleteRouteUsingSupportNetInterface(dest, gw, c.Support.V4CIDR)
	} else {
		gw, ok = FindHostIPForNAT64(c)
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

	if !node.IsNAT64Server && !node.IsDNS64Server {
		dest = c.Support.CIDR
		gw, ok = FindHostIPForNAT64(c)
		if !ok {
			err = fmt.Errorf("Unable to find node with NAT64 server configured")
		} else {
			err = DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
		}
		if err != nil {
			glog.Warningf("Unable to delete route to %s via %s: %s", dest, gw, err.Error())
		} else {
			glog.V(4).Infof("Deleted route to %s via %s", dest, gw)
		}
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

	d := filepath.Join(c.General.WorkArea, DNS64BaseArea)
	err = os.RemoveAll(d)
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

func Cleanup(name string, c *Config) {
	node := c.Topology[name]
	glog.Infof("Cleaning %q", name)
	if node.IsMaster || node.IsMinion {
		CleanupClusterNode(&node, c)
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
	glog.Infof("Node %q cleaned", name)
}
