package lazyjack

import (
	"fmt"
	"os"

	"github.com/golang/glog"
)

// CleanupForPlugin performs actions to cleanup the CNI plugin on a node,
// Including removing the CNI config file and area.
func CleanupForPlugin(node *Node, c *Config) error {
	glog.V(1).Infof("Cleaning up for %s plugin", c.General.Plugin)

	err := c.General.CNIPlugin.Cleanup(node)
	if err != nil {
		return err
	}

	// Note: CNI config file will be removed, when "kubeadm reset" performed
	err = os.RemoveAll(c.General.CNIArea)
	if err != nil {
		return fmt.Errorf("unable to remove CNI config file and area: %v", err)
	}
	glog.V(4).Info("removed CNI config file and area")
	glog.Infof("Cleaned up for %s plugin", c.General.Plugin)
	return nil
}

// StopKubernetes is called during the "down" operation, to bring down
// the cluster.
func StopKubernetes() error {
	args := []string{"reset", "-f"}
	output, err := DoExecCommand("kubeadm", args)
	if err != nil {
		return fmt.Errorf("unable to %s Kubernetes cluster: %v", args[0], err)
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

	glog.Infof("Node %q tore down", name)
}
