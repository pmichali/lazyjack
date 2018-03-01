package lazyjack_test

import (
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

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
		General: lazyjack.GeneralSettings{
			Token:         "<valid-token-here>",
			TokenCertHash: "<valid-ca-certificate-hash-here>",
		},
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
	expected := []string{"init", "--config=/tmp/lazyjack/kubeadm.conf"}

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
