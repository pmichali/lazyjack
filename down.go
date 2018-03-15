package lazyjack

import (
	"fmt"
	"os"

	"github.com/golang/glog"
)

func CleanupForPlugin(node *Node, c *Config) error {
	glog.V(1).Infof("Cleaning up for %s plugin", c.General.Plugin)

	err := RemoveRoutesForPodNetwork(node, c)
	if err != nil {
		return fmt.Errorf("Unable to remove routes for %s plugin: %s", c.General.Plugin, err.Error())
	}
	glog.V(1).Infof("Removed routes for %s plugin", c.General.Plugin)

	// Note: CNI config file will be removed, when "kubeadm reset" performed
	err = os.RemoveAll(c.General.CNIArea)
	if err != nil {
		return fmt.Errorf("Unable to remove CNI config file and area: %s", err.Error())
	}
	glog.V(1).Info("Removed CNI config file and area")
	glog.Infof("Cleaned up for %s plugin", c.General.Plugin)
	return nil
}

func StopKubernetes() error {
	args := []string{"reset"}
	output, err := DoExecCommand("kubeadm", args)
	if err != nil {
		glog.Warningf("Unable to %s Kubernetes cluster: %s", args[0], err.Error())
		os.Exit(1)
	}
	glog.V(1).Infof("Kubernetes %s output: %s", args[0], output)
	glog.Infof("Kubernetes %s successful", args[0])
	return nil
}

func (n *NetManager) RemoveBridge(name string) error {
	glog.V(1).Infof("Removing bridge %q", name)
	err := n.BringLinkDown(name)
	if err == nil {
		glog.Infof("Brought link %q down", name)
	}
	// Even if err, will try to delete bridge
	err2 := n.DeleteLink(name)
	if err2 == nil {
		glog.Infof("Removed bridge %q", name)
	}
	if err == nil {
		return err2
	} else if err2 == nil {
		return err
	}
	return fmt.Errorf("Unable to bring link down (%s), nor remove link (%s)", err.Error(), err2.Error())
}

func TearDown(name string, c *Config) {
	node := c.Topology[name]
	var asType string
	switch {
	case node.IsMaster:
		asType = "master"
	case node.IsMinion:
		asType = "minion"
	default:
		glog.Infof("Skipping - node %q role is not master or minion", name)
		return
	}
	glog.Infof("Tearing down %q as %s", name, asType)

	err := StopKubernetes()
	if err != nil {
		glog.Warningf("Unable to reset cluster: %s", err.Error())
	}

	// Leave kubeadm.conf, in case user customized it.

	err = CleanupForPlugin(&node, c)
	if err != nil {
		glog.Warningf(err.Error())
	}

	err = c.General.NetMgr.RemoveBridge("br0")
	if err != nil {
		glog.Warningf("Unable to remove br0 bridge: %s", err.Error())
	}

	glog.Infof("Node %q tore down", name)
}
