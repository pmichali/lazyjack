package orca

import (
	"os"

	"github.com/golang/glog"
)

func CleanupForPlugin(node *Node, c *Config) {
	glog.V(1).Infof("Cleaning up for %s plugin", c.Plugin)

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
	glog.Infof("Cleaned up for %s plugin", c.Plugin)
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
		glog.Infof("Skipping node %q as role is not master or minion", name)
		return
	}
	glog.V(1).Infof("Tearing down %q as %s", name, asType)

	// Do kubeadm reset
	CleanupForPlugin(&node, c)

	glog.Infof("Node %q tore down", name)
}
