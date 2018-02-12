package orca_test

import (
	"orca"
	"testing"
)

func TestBuildNodeCIDR(t *testing.T) {
	actual := orca.BuildNodeCIDR("2001:db8:20::", 2, 64)
	expected := "2001:db8:20::2/64"
	if actual != expected {
		t.Errorf("FAILED: Node CIDR create. Expected %q, got %q", expected, actual)
	}
}

func TestBuildDestCIDR(t *testing.T) {
	actual := orca.BuildDestCIDR("2001:db8", 20, 80)
	expected := "2001:db8:20::/80"
	if actual != expected {
		t.Errorf("FAILED: Destination CIDR create. Expected %q, got %q", expected, actual)
	}
}

func TestBuildGWIP(t *testing.T) {
	actual := orca.BuildGWIP("2001:db8::", 5)
	expected := "2001:db8::5"
	if actual != expected {
		t.Errorf("FAILED: Gateway IP create. Expected %q, got %q", expected, actual)
	}
}
