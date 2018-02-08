package orca_test

import (
	"orca"
	"strings"
	"testing"
)

func TestBuildDockerArgsForDNS64(t *testing.T) {
	c := &orca.Config{
		DNS64: orca.DNS64Config{ServerIP: "2001:db8::100"},
	}

	list := orca.BuildRunArgsForDNS64(c)
	actual := strings.Join(list, " ")
	expected := "run -d --name bind9 --hostname bind9 --label orcha --privileged=true --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.all.forwarding=1 --ip6 2001:db8::100 --dns 2001:db8::100 -v /tmp/bind9/conf/named.conf:/etc/bind/named.conf --net support_net resystit/bind9:latest"
	if actual != expected {
		t.Errorf("FAILED: Docker run command for DNS. Expected %q, got %q", expected, actual)
	}
}
