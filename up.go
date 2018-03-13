package lazyjack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/golang/glog"
)

func EnsureCNIAreaExists(area string) error {
	err := os.RemoveAll(area)
	if err != nil {
		return err
	}
	err = os.MkdirAll(area, 0755)
	if err != nil {
		return err
	}
	return nil
}

func SetupForPlugin(node *Node, c *Config) error {
	glog.V(1).Infof("Setting up %s plugin", c.General.Plugin)
	err := EnsureCNIAreaExists(c.General.CNIArea)
	if err != nil {
		return err
	} else {
		glog.V(4).Info("Created area for CNI config file")
	}
	err = CreateBridgeCNIConfigFile(node, c)
	if err != nil {
		return err
	}

	err = CreateRoutesForPodNetwork(node, c)
	if err != nil {
		return err
	}
	glog.Infof("Set up for %s plugin", c.General.Plugin)
	return nil
}

func RestartKubeletService() error {
	_, err := DoExecCommand("systemctl", []string{"daemon-reload"})
	if err != nil {
		glog.Fatalf("Unable to reload daemons: %s", err.Error())
		os.Exit(1)
	}
	glog.V(1).Info("Reloaded daemons")

	_, err = DoExecCommand("systemctl", []string{"restart", "kubelet"})
	if err != nil {
		glog.Fatalf("Unable to restart kubelet service: %s", err.Error())
		os.Exit(1)
	}
	glog.V(1).Info("Restarted kubelet service")
	glog.Info("Reloaded daemons and restarted kubelet service")
	return nil
}

func BuildKubeAdmCommand(n, master *Node, c *Config) []string {
	var args []string
	if n.IsMaster {
		file := filepath.Join(c.General.WorkArea, KubeAdmConfFile)
		args = []string{"init", fmt.Sprintf("--config=%s", file)}
	} else {
		args = []string{
			"join",
			"--token", c.General.Token,
			fmt.Sprintf("[%s%d]:6443", c.Mgmt.Prefix, master.ID),
			// "--discovery-token-unsafe-skip-ca-verification",
			"--discovery-token-ca-cert-hash",
			fmt.Sprintf("sha256:%s", c.General.TokenCertHash),
		}
	}
	return args
}

func CopyFile(name, src, dst string) (err error) {
	glog.V(4).Infof("Copying %s/%s to %s/%s", src, name, dst, name)
	s, err := os.Open(filepath.Join(src, name))
	if err != nil {
		return fmt.Errorf("Unable to open source file %q: %s", name, err.Error())
	}
	defer s.Close()

	d, err := os.Create(filepath.Join(dst, name))
	if err != nil {
		return fmt.Errorf("Unable to open destination file %q: %s", name, err.Error())
	}
	defer func() {
		cerr := d.Close()
		if err == nil && cerr != nil {
			err = fmt.Errorf("Unable to close destination file %q: %s", name, cerr.Error())
		}
	}()

	_, err = io.Copy(d, s)
	if err != nil {
		return fmt.Errorf("Unable to copy %q from %q to %q: %s", name, src, dst, err.Error())
	}
	err = d.Sync()
	if err != nil {
		return fmt.Errorf("Unable to flush data to destination file %q: %s", name, err.Error())
	}
	return
}

func PlaceCertificateAndKeyForCA(workBase, dst string) error {
	glog.V(1).Infof("Copying certificate and key to Kuberentes area")
	src := filepath.Join(workBase, CertArea)
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create area for Kubernetes certificates (%s): %s", dst, err.Error())
	}
	err = CopyFile("ca.crt", src, dst)
	if err != nil {
		return err
	}
	err = CopyFile("ca.key", src, dst)
	if err == nil {
		glog.Infof("Copied certificate and key to Kuberentes area")
	}
	return err
}

func DetermineMasterNode(c *Config) *Node {
	for _, node := range c.Topology {
		if node.IsMaster {
			return &node
		}
	}
	return nil
}

func StartKubernetes(n *Node, c *Config) error {
	master := DetermineMasterNode(c)
	if master == nil {
		return fmt.Errorf("Unable to determine master node")
	}

	args := BuildKubeAdmCommand(n, master, c)

	glog.Infof("Starting Kubernetes on %s... (please wait)", n.Name)
	output, err := DoExecCommand("kubeadm", args)
	if err != nil {
		glog.Fatalf("Unable to %s Kubernetes cluster: %s", args[0], err.Error())
		os.Exit(1)
	}
	glog.V(1).Info("Kubernetes %s output: %s", args[0], output)
	glog.Infof("Kubernetes %s successful", args[0])
	return nil
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

	err := SetupForPlugin(&node, c)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1) // TODO: Rollback?
	}

	err = RestartKubeletService()
	if err != nil {
		glog.Fatalf(err.Error())
		os.Exit(1) // TODO: Rollback?
	}

	if node.IsMaster {
		err = PlaceCertificateAndKeyForCA(c.General.WorkArea, c.General.K8sCertArea)
		if err != nil {
			glog.Fatalf(err.Error())
			os.Exit(1) // TODO: Rollback?
		}
	}

	err = StartKubernetes(&node, c)
	if err != nil {
		glog.Fatalf(err.Error())
		os.Exit(1) // TODO: Rollback?
	}

	// FUTURE: update ~/.kube/config (how to know user?)

	glog.Infof("Node %q brought up", name)
}
