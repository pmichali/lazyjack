package orca

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

func CreateKubeAdmConfigContents(n *Node, c *Config) *bytes.Buffer {
	header := `# Autogenerated file
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
kubernetesVersion: 1.9.0
api:
`
	trailer := `tokenTTL: 0s
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
  runtime-config: "admissionregistration.k8s.io/v1alpha1"
  feature-gates: AllAlpha=true
`
	contents := bytes.NewBufferString(header)
	fmt.Fprintf(contents, "  advertiseAddress: \"%s%d\"\n", c.Mgmt.Subnet, n.ID)
	fmt.Fprintf(contents, "networking:\n")
	fmt.Fprintf(contents, "  serviceSubnet: %q\n", c.Service.CIDR)
	fmt.Fprintf(contents, "nodeName: %s\n", n.Name)
	fmt.Fprintf(contents, "token: %q\n", c.Token)
	fmt.Fprintf(contents, trailer)
	return contents
}

func CreateKubeAdmConfigFile(node *Node, c *Config) error {
	contents := CreateKubeAdmConfigContents(node, c)

	err := ioutil.WriteFile(KubeAdmConfFile, contents.Bytes(), 0755)
	if err == nil {
		glog.V(1).Infof("Created %s file", KubeAdmConfFile)
	}
	return err
}

func BuildKubeAdmCommand(n, master *Node, c *Config) []string {
	var args []string
	if n.IsMaster {
		args = []string{"init", fmt.Sprintf("--config=%s", KubeAdmConfFile)}
	} else {
		args = []string{
			"join",
			"--token", c.Token,
			fmt.Sprintf("[%s%d]:6443", c.Mgmt.Subnet, master.ID),
			// "--discovery-token-unsafe-skip-ca-verification",
			"--discovery-token-ca-cert-hash",
			fmt.Sprintf("sha256:%s", c.TokenCertHash),
		}
	}
	return args
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

	output, err := DoExecCommand("kubeadm", args)
	if err != nil {
		glog.Fatalf("Unable to %s Kubernetes cluster: %s", args[0], err.Error())
		os.Exit(1)
	}
	glog.V(1).Info("Kubernetes %s output: %s", args[0], output)
	glog.Info("Kubernetes %s successful", args[0])
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

	SetupForPlugin(&node, c)

	err := RestartKubeletService()
	if err != nil {
		glog.Fatalf(err.Error())
		os.Exit(1) // TODO: Rollback?
	}

	if node.IsMaster {
		err = CreateKubeAdmConfigFile(&node, c)
		if err != nil {
			glog.Fatalf(err.Error())
			os.Exit(1) // TODO: Rollback?
		}
		// TODO: Copy CA cert and key to /etc/kubernetes/pki/
	}

	err = StartKubernetes(&node, c)
	if err != nil {
		glog.Fatalf(err.Error())
		os.Exit(1) // TODO: Rollback?
	}

	// FUTURE: update ~/.kube/config (how to know user?)

	glog.Infof("Node %q brought up", name)
}
