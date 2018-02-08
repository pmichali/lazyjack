package orca_test

import (
	"orca"
	"testing"
)

func TestKubeletDropInContents(t *testing.T) {
	c := &orca.Config{
		DNS64: orca.DNS64Config{ServerIP: "2001:db8::100"},
	}

	expected := `[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=2001:db8::100 --cluster-domain=cluster.local"
`
	actual := orca.CreateKubeletDropInContents(c)
	if actual.String() != expected {
		t.Errorf("Kubelet drop-in contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}
