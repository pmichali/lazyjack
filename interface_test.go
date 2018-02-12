package orca_test

import (
	"orca"
	"testing"
)

func TestCreateCIDR(t *testing.T) {
	actual := orca.BuildCIDR("2001:db8:20::", 2, 64)
	expected := "2001:db8:20::2/64"
	if actual != expected {
		t.Errorf("FAILED: CIDR create. Expected %q, got %q", expected, actual)
	}
}
