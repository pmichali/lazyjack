package orca_test

import (
	"orca"
	"strings"
	"testing"
)

func TestCreateCIDR(t *testing.T) {
	actual := orca.BuildCIDR("2001:db8:20::", 2, 64)
	expected := "2001:db8:20::2/64"
	if actual != expected {
		t.Errorf("FAILED: CIDR create. Expected %q, got %q", expected, actual)
	}
}

func TestBuildIpRouteAddArgs(t *testing.T) {
	list := orca.BuildIpRouteAddArgs("172.18.0.128/25", "172.18.0.200")
	actual := strings.Join(list, " ")
	expected := "route add 172.18.0.128/25 via 172.18.0.200"
	if actual != expected {
		t.Errorf("FAILED: Route add args wrong. Expected %q, got %q", expected, actual)
	}
}

func TestBuildIpRouteDeleteArgs(t *testing.T) {
	list := orca.BuildIpRouteDeleteArgs("172.18.0.128/25", "172.18.0.200")
	actual := strings.Join(list, " ")
	expected := "route del 172.18.0.128/25 via 172.18.0.200"
	if actual != expected {
		t.Errorf("FAILED: Route delete args wrong. Expected %q, got %q", expected, actual)
	}
}
