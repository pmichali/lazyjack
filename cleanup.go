package lazyjack

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		return fmt.Errorf("Unable to read file %s to revert: %s", file, err.Error())
	}
	contents = RevertConfigInfo(contents, file)
	err = SaveFileContents(contents, file, backup)
	if err != nil {
		return err
	}
	glog.V(4).Infof("Restored %s contents", file)
	return nil
}

func RemoveDropInFile(c *Config) error {
	glog.V(1).Info("Cleaning kubelet drop-in file")
	file := filepath.Join(c.General.SystemdArea, KubeletDropInFile)
	err := os.Remove(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("No kubelet drop-in file to remove")
		}
		return fmt.Errorf("Unable to remove kubelet drop-in file (%s): %s", file, err.Error())
	}
	glog.V(4).Info("Removed kubelet drop-in file")
	return nil
}

func RemoveManagementIP(node *Node, c *Config) error {
	mgmtIP := BuildNodeCIDR(c.Mgmt.Prefix, node.ID, c.Mgmt.Size)
	err := c.General.NetMgr.RemoveAddressFromLink(mgmtIP, node.Interface)
	if err != nil {
		return fmt.Errorf("Unable to remove IP from management interface: %s", err.Error())
	}
	glog.V(4).Info("Removed IP address from management interface")
	return nil
}

func RevertEtcAreaFile(c *Config, file, backup string) error {
	file = filepath.Join(c.General.EtcArea, file)
	backup = filepath.Join(c.General.EtcArea, backup)
	err := RevertEntries(file, backup)
	if err == nil {
		glog.V(4).Infof("Reverted %q with backup %q", file, backup)
	}
	return err
}

// RemoveRouteForDNS64 removes the route to the DNS64 network via the
// IPv4 support network, if on the NAT64 node, or via the NAT64 server's
// management IP, if not on the NAT64 server.
func RemoveRouteForDNS64(node *Node, c *Config) error {
	dest := c.DNS64.CIDR
	var gw string
	var ok bool
	var err error
	if node.IsNAT64Server {
		gw = c.NAT64.ServerIP
		err = c.General.NetMgr.DeleteRouteUsingSupportNetInterface(dest, gw, c.Support.V4CIDR)
	} else {
		gw, ok = FindHostIPForNAT64(c)
		if !ok {
			err = fmt.Errorf("Unable to find node with NAT64 server")
		} else {
			err = c.General.NetMgr.DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
		}
	}
	if err != nil {
		return fmt.Errorf("Unable to delete route to %s via %s: %s", dest, gw, err.Error())
	}
	glog.V(4).Infof("Deleted route to %s via %s", dest, gw)
	return nil
}

// RemoveRouteForNAT64 removes the route to the support network via the
// NAT64 server's manage,ent IP.
func RemoveRouteForNAT64(node *Node, c *Config) error {
	dest := c.Support.CIDR
	var gw string
	var ok bool
	var err error
	gw, ok = FindHostIPForNAT64(c)
	if !ok {
		err = fmt.Errorf("Unable to find node with NAT64 server configured")
	} else {
		err = c.General.NetMgr.DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
	}
	if err != nil {
		return fmt.Errorf("Unable to delete route to %s via %s: %s", dest, gw, err.Error())
	}
	glog.V(4).Infof("Deleted route to %s via %s", dest, gw)
	return nil
}

func CleanupClusterNode(node *Node, c *Config) error {
	var all []string
	glog.V(1).Info("Cleaning general settings")
	err := RemoveDropInFile(c)
	if err != nil {
		glog.V(4).Info(err.Error())
		all = append(all, err.Error())
	}

	err = RemoveManagementIP(node, c)
	if err != nil {
		glog.V(4).Info(err.Error())
		all = append(all, err.Error())
	}

	err = RevertEtcAreaFile(c, EtcHostsFile, EtcHostsBackupFile)
	if err != nil {
		glog.V(4).Info(err.Error())
		all = append(all, err.Error())
	}
	err = RevertEtcAreaFile(c, EtcResolvConfFile, EtcResolvConfBackupFile)
	if err != nil {
		glog.V(4).Info(err.Error())
		all = append(all, err.Error())
	}

	err = RemoveRouteForDNS64(node, c)
	if err != nil {
		glog.V(4).Info(err.Error())
		all = append(all, err.Error())
	}

	if !node.IsNAT64Server && !node.IsDNS64Server {
		err = RemoveRouteForNAT64(node, c)
		if err != nil {
			glog.V(4).Info(err.Error())
			all = append(all, err.Error())
		}
	}
	glog.Info("Cleaned general settings")
	if len(all) > 0 {
		return fmt.Errorf(strings.Join(all, ". "))
	}
	return nil
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

	err := c.General.NetMgr.DeleteRouteUsingSupportNetInterface(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		glog.Warning(err.Error())
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

func CleanupSupportNetwork() error {
	if !ResourceExists(SupportNetName) {
		return fmt.Errorf("Skipping - support network does not exists")
	}

	args := BuildDeleteNetArgsForSupportNet()
	_, err := DoCommand("SupportNetName", args)
	if err != nil {
		return fmt.Errorf("Unable to remove support network")
	}
	glog.Info("Cleaned support network")
	return nil
}

func Cleanup(name string, c *Config) {
	node := c.Topology[name]
	var err error
	glog.Infof("Cleaning %q", name)
	if node.IsMaster || node.IsMinion {
		err = CleanupClusterNode(&node, c)
		if err != nil {
			glog.Warning(err.Error())
		}
	}
	if node.IsDNS64Server {
		CleanupDNS64Server(&node, c)
	}
	if node.IsNAT64Server {
		CleanupNAT64Server(&node, c)
	}
	if node.IsDNS64Server || node.IsNAT64Server {
		err = CleanupSupportNetwork()
		if err != nil {
			glog.Warning(err.Error())
		}
	}
	glog.Infof("Node %q cleaned", name)
}
