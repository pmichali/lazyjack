package lazyjack_test

import (
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestKubeAdmConfigContents(t *testing.T) {
	c := &lazyjack.Config{
		Token: "56cdce.7b18ad347f3de81c",
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# Autogenerated file
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
kubernetesVersion: 1.9.0
api:
  advertiseAddress: "fd00:100::10"
networking:
  serviceSubnet: "fd00:30::/110"
nodeName: my-master
token: "56cdce.7b18ad347f3de81c"
tokenTTL: 0s
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
  runtime-config: "admissionregistration.k8s.io/v1alpha1"
  feature-gates: AllAlpha=true
`
	actual := lazyjack.CreateKubeAdmConfigContents(n, c)
	if actual.String() != expected {
		t.Errorf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func SlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildKubeAdmCommand(t *testing.T) {
	c := &lazyjack.Config{
		Token:         "<valid-token-here>",
		TokenCertHash: "<valid-ca-certificate-hash-here>",
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	minionNode := &lazyjack.Node{
		ID:       20,
		IsMaster: false,
	}
	masterNode := &lazyjack.Node{
		ID:       10,
		IsMaster: true,
	}
	actual := lazyjack.BuildKubeAdmCommand(masterNode, masterNode, c)
	expected := []string{"init", "--config=/tmp/kubeadm.conf"}

	if !SlicesEqual(actual, expected) {
		t.Errorf("KubeAdm init args incorrect for master node. Expected %q, got %q", strings.Join(expected, " "), strings.Join(actual, " "))
	}

	actual = lazyjack.BuildKubeAdmCommand(minionNode, masterNode, c)
	expected = []string{"join", "--token", "<valid-token-here>",
		//	"[fd00:100::10]:6443", "--discovery-token-unsafe-skip-ca-verification"}
		"[fd00:100::10]:6443", "--discovery-token-ca-cert-hash", "sha256:<valid-ca-certificate-hash-here>"}

	if !SlicesEqual(actual, expected) {
		t.Errorf("KubeAdm init args incorrect for master node. Expected %q, got %q", strings.Join(expected, " "), strings.Join(actual, " "))
	}
}
