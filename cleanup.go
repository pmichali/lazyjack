package lazyjack

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

// RevertConfigInfo restores the contents of the provided config file,
// by using the comment tags that describe the additions settings (to be
// removed), and the previous settings (to be restored).
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

// RevertEntries obtains the config file contents, reverts the
// settings, and then stores the updated file (with a backup
// created, in case of issues).
func RevertEntries(file, backup string) error {
	glog.V(4).Infof("Cleaning %s file", file)
	contents, err := GetFileContents(file)
	if err != nil {
		return fmt.Errorf("unable to read file %s to revert: %v", file, err)
	}
	contents = RevertConfigInfo(contents, file)
	err = SaveFileContents(contents, file, backup)
	if err != nil {
		return err
	}
	glog.V(4).Infof("Restored %s contents", file)
	return nil
}

// RemoveDropInFile eliminates the kubelet drop-in file as part of
// the cleanup operation.
func RemoveDropInFile(c *Config) error {
	glog.V(1).Info("Cleaning kubelet drop-in file")
	file := filepath.Join(c.General.SystemdArea, KubeletDropInFile)
	err := os.Remove(file)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no kubelet drop-in file to remove")
		}
		return fmt.Errorf("unable to remove kubelet drop-in file (%s): %s", file, err.Error())
	}
	glog.V(4).Info("Removed kubelet drop-in file")
	return nil
}

// RemoveManagementIP removes the node's management IP off of the
// interface configured as the management port.
func RemoveManagementIP(node *Node, c *Config) error {
	mgmtIP := BuildNodeCIDR(c.Mgmt.Info[0], node.ID)
	err := c.General.NetMgr.RemoveAddressFromLink(mgmtIP, node.Interface)
	if err != nil {
		return fmt.Errorf("unable to remove IP from management interface: %v", err)
	} else {
		glog.V(4).Infof("removed %s from %s", mgmtIP, node.Interface)
	}
	if c.General.Mode == DualStackNetMode {
		mgmtIP = BuildNodeCIDR(c.Mgmt.Info[1], node.ID)
		err = c.General.NetMgr.RemoveAddressFromLink(mgmtIP, node.Interface)
		if err != nil {
			return fmt.Errorf("unable to remove second IP from management interface: %v", err)
		} else {
			glog.V(4).Infof("Removed %s from %s", mgmtIP, node.Interface)
		}
	}
	return nil
}

// RevertEtcAreaFile will revert the entries in config files in
// the /etc/ area as part of cleanup.
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
			err = fmt.Errorf("unable to find node with NAT64 server")
		} else {
			err = c.General.NetMgr.DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
		}
	}
	if err != nil {
		return fmt.Errorf("unable to delete route to %s via %s: %v", dest, gw, err)
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
		err = fmt.Errorf("unable to find node with NAT64 server configured")
	} else {
		err = c.General.NetMgr.DeleteRouteUsingInterfaceName(dest, gw, node.Interface)
	}
	if err != nil {
		return fmt.Errorf("unable to delete route to %s via %s: %v", dest, gw, err)
	}
	glog.V(4).Infof("Deleted route to %s via %s", dest, gw)
	return nil
}

// CleanupClusterNode removes the kubelet drop-in file, management port's
// IP address, reverts /etc/hosts and /etc/resolv.conf, removes route for
// DNS network, and removes route to NAT64 server (when the node is not
// the node hosting the server).
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

	if c.General.Mode == "ipv6" {
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
	}

	glog.Info("Cleaned general settings")
	if len(all) > 0 {
		return fmt.Errorf(strings.Join(all, ". "))
	}
	return nil
}

// RemoveContainer checks to see if the container is present, and if so,
// removes the container.
func RemoveContainer(name string, c *Config) error {
	if c.General.Hyper.ResourceState(name) == ResourceNotPresent {
		return fmt.Errorf("skipping - No %q container exists", name)
	}
	err := c.General.Hyper.DeleteContainer(name)
	if err != nil {
		return fmt.Errorf("unable to remove %q container: %v", name, err)
	}
	glog.V(4).Infof("Removed %q container", name)
	return nil
}

// CleanupDNS64Server removes the DNS64 server and associated config
// files. The IPv4 default route is not altered.
func CleanupDNS64Server(c *Config) error {
	glog.V(1).Info("Cleaning DNS64")
	var all []string
	var err error
	err = RemoveContainer(DNS64Name, c)
	if err != nil {
		all = append(all, err.Error())
	} else {
		glog.V(4).Info("Removed DNS64 container")
	}

	if c.General.Hyper.ResourceState(DNS64Volume) == ResourceNotPresent {
		all = append(all, fmt.Sprintf("skipping - No %q volume", DNS64Volume))
	} else {
		err = c.General.Hyper.DeleteVolume(DNS64Volume)
		if err != nil {
			all = append(all, err.Error())
		} else {
			glog.V(4).Info("Removed volume used for DNS64 container")
		}
	}

	// Will leave V4 default route

	glog.Info("Cleaned DNS64")
	if len(all) > 0 {
		return fmt.Errorf(strings.Join(all, ". "))
	}
	return nil
}

// CleanupNAT64Server removes the NAT64 server and route to server.
// The default IPv4 route is not touched.
func CleanupNAT64Server(c *Config) error {
	glog.V(1).Info("Cleaning NAT64")

	var all []string
	var err error
	err = c.General.NetMgr.DeleteRouteUsingSupportNetInterface(c.NAT64.V4MappingCIDR, c.NAT64.V4MappingIP, c.Support.V4CIDR)
	if err != nil {
		all = append(all, err.Error())
	} else {
		glog.V(1).Info("Removed local IPv4 route to NAT64 container")
	}

	err = RemoveContainer(NAT64Name, c)
	if err != nil {
		all = append(all, err.Error())
	} else {
		glog.V(4).Info("Removed NAT64 container")
	}

	// Will leave default V4 route

	glog.Info("Cleaned NAT64")
	if len(all) > 0 {
		return fmt.Errorf(strings.Join(all, ". "))
	}
	return nil
}

// CleanupSupportNetwork checks to see if the support network exists,
// and if so, removes the network.
func CleanupSupportNetwork(c *Config) error {
	if c.General.Hyper.ResourceState(SupportNetName) == ResourceNotPresent {
		return fmt.Errorf("skipping - support network does not exists")
	}

	err := c.General.Hyper.DeleteNetwork(SupportNetName)
	if err != nil {
		return fmt.Errorf("unable to remove support network: %v", err)
	}
	glog.Info("Cleaned support network")
	return nil
}

// Cleanup is the top level method for the "clean" action, to remove/revert
// config files, remove routes, and delete DNS64/NAT64 containers.
func Cleanup(name string, c *Config) error {
	node := c.Topology[name]
	var all []string
	var err error
	glog.Infof("Cleaning %q", name)
	if node.IsMaster || node.IsMinion {
		err = CleanupClusterNode(&node, c)
		if err != nil {
			all = append(all, err.Error())
		}
	}
	if c.General.Mode == IPv6NetMode {
		if node.IsDNS64Server {
			err = CleanupDNS64Server(c)
			if err != nil {
				all = append(all, err.Error())
			}
		}
		if node.IsNAT64Server {
			err = CleanupNAT64Server(c)
			if err != nil {
				all = append(all, err.Error())
			}
		}
		if node.IsDNS64Server || node.IsNAT64Server {
			err = CleanupSupportNetwork(c)
			if err != nil {
				all = append(all, err.Error())
			}
		}
	}

	glog.Infof("Node %q cleaned", name)
	if len(all) > 0 {
		return fmt.Errorf(strings.Join(all, ". "))
	}
	return nil
}
