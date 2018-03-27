package lazyjack

import (
	"fmt"
	"os"

	"github.com/golang/glog"
)

// CleanupForPlugin removes all static routes from the node to other
// nodes and removes the bridge CNI plugin config file to clean up the node.
func CleanupForPlugin(node *Node, c *Config) error {
	glog.V(1).Infof("Cleaning up for %s plugin", c.General.Plugin)

	err := RemoveRoutesForPodNetwork(node, c)
	if err != nil {
		return fmt.Errorf("unable to remove routes for %s plugin: %v", c.General.Plugin, err)
	}
	glog.V(1).Infof("Removed routes for %s plugin", c.General.Plugin)

	// Note: CNI config file will be removed, when "kubeadm reset" performed
	err = os.RemoveAll(c.General.CNIArea)
	if err != nil {
		return fmt.Errorf("unable to remove CNI config file and area: %v", err)
	}
	glog.V(1).Info("Removed CNI config file and area")
	glog.Infof("Cleaned up for %s plugin", c.General.Plugin)
	return nil
}

// StopKubernetes is called during the "down" operation, to bring down
// the cluster.
func StopKubernetes() error {
	args := []string{"reset"}
	output, err := DoExecCommand("kubeadm", args)
	if err != nil {
		glog.Warningf("unable to %s Kubernetes cluster: %v", args[0], err)
		os.Exit(1)
	}
	glog.V(1).Infof("Kubernetes %s output: %s", args[0], output)
	glog.Infof("Kubernetes %s successful", args[0])
	return nil
}

// TearDown performs the "down" operations of bringing down the cluster,
// removing static routes, removing the Bridge plugin config file, and
// removing the bridge.
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
		glog.Warningf("unable to reset cluster: %v", err)
	}

	// Leave kubeadm.conf, in case user customized it.

	err = CleanupForPlugin(&node, c)
	if err != nil {
		glog.Warningf(err.Error())
	}

	err = c.General.NetMgr.RemoveBridge("br0")
	if err != nil {
		glog.Warningf("unable to remove br0 bridge: %v", err)
	}

	glog.Infof("Node %q tore down", name)
}
