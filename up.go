package orca

import (
	"os"

	"github.com/golang/glog"
)

func EnsureCNIAreaExists() error {
	err := os.RemoveAll(CNIConfArea)
	if err != nil {
		return err
	}
	err = os.MkdirAll(CNIConfArea, 0755)
	if err != nil {
		return err
	}
	return nil
}

func SetupForPlugin(node *Node, c *Config) {
	glog.V(1).Infof("Setting up %s plugin", c.Plugin)
	err := EnsureCNIAreaExists()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	} else {
		glog.V(4).Info("Created area for CNI config file")
	}
	err = CreateBridgeCNIConfigFile(node, c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	err = CreateRoutesForPodNetwork(node, c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}
	glog.Infof("Set up for %s plugin", c.Plugin)
}

func BringUp(name string, c *Config) {
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
	glog.V(1).Infof("Bringing up %q as %s", name, asType)

	SetupForPlugin(&node, c)
	// Reload daemns and restart kubelet
	// Run kubeadm init/join
	// [master] update ~/.kube/config

	glog.Infof("Node %q brought up", name)
}
