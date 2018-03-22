package lazyjack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	}
	glog.V(4).Info("Created area for CNI config file")
	err = CreateBridgeCNIConfigFile(node, c)
	if err != nil {
		return err
	}

	err = CreateRoutesForPodNetwork(node, c)
	if err != nil {
		// Note: May get error, if route already exists. Since this is the
		// last operation, it is OK to return, versus continuing here.
		return err
	}
	glog.Infof("Set up for %s plugin", c.General.Plugin)
	return nil
}

func RestartKubeletService() error {
	_, err := DoExecCommand("systemctl", []string{"daemon-reload"})
	if err != nil {
		glog.Fatalf("unable to reload daemons: %v", err)
		os.Exit(1)
	}
	glog.V(1).Info("Reloaded daemons")

	_, err = DoExecCommand("systemctl", []string{"restart", "kubelet"})
	if err != nil {
		glog.Fatalf("unable to restart kubelet service: %v", err)
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
		return fmt.Errorf("unable to open source file %q: %v", name, err)
	}
	defer s.Close()

	d, err := os.Create(filepath.Join(dst, name))
	if err != nil {
		return fmt.Errorf("unable to open destination file %q: %v", name, err)
	}
	defer func() {
		cerr := d.Close()
		if err == nil && cerr != nil {
			err = fmt.Errorf("unable to close destination file %q: %v", name, cerr)
		}
	}()

	_, err = io.Copy(d, s)
	if err != nil {
		return fmt.Errorf("unable to copy %q from %q to %q: %v", name, src, dst, err)
	}
	err = d.Sync()
	if err != nil {
		return fmt.Errorf("unable to flush data to destination file %q: %v", name, err)
	}
	return
}

func PlaceCertificateAndKeyForCA(workBase, dst string) error {
	glog.V(1).Infof("Copying certificate and key to Kuberentes area")
	src := filepath.Join(workBase, CertArea)
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return fmt.Errorf("unable to create area for Kubernetes certificates (%s): %v", dst, err)
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
		return fmt.Errorf("unable to determine master node")
	}

	args := BuildKubeAdmCommand(n, master, c)

	glog.Infof("Starting Kubernetes on %s... (please wait)", n.Name)
	output, err := DoExecCommand("kubeadm", args)
	if err != nil {
		glog.Fatalf("unable to %s Kubernetes cluster: %v", args[0], err)
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
		if strings.HasPrefix(err.Error(), "skipping -") {
			glog.Warning(err.Error())
			// Will keep going...
		} else {
			glog.Fatal(err.Error())
			os.Exit(1) // TODO: Rollback?
		}
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
