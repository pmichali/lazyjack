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

func TestNamedConfContents(t *testing.T) {
	c := &orca.Config{
		DNS64: orca.DNS64Config{
			Prefix:         "fd00:10:64:ff9b::",
			PrefixSize:     96,
			RemoteV4Server: "8.8.8.8",
		},
	}

	expected := `options {
    directory "/var/bind";
    allow-query { any; };
    forwarders {
        fd00:10:64:ff9b::8.8.8.8;
    };
    auth-nxdomain no;    # conform to RFC1035
    listen-on-v6 { any; };
    dns64 fd00:10:64:ff9b::/96 {
        exclude { any; };
    };
};
`
	actual := orca.CreateNamedConfContents(c)
	if actual.String() != expected {
		t.Errorf("DNS64 named.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func TestParseIPv4AddressFromIfConfig(t *testing.T) {
	var testCases = []struct {
		name     string
		ifConfig string
		expected string
	}{
		{
			name: "v4 address found",
			ifConfig: `39139: eth0@if39140: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
    link/ether 02:42:ac:12:00:02 brd ff:ff:ff:ff:ff:ff
    inet 172.18.0.2/16 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fd00:10::100/64 scope global flags 02
       valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:2/64 scope link
       valid_lft forever preferred_lft forever`,
			expected: "172.18.0.2/16",
		},
		{
			name: "no ipv4 address",
			ifConfig: `39139: eth0@if39140: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
    link/ether 02:42:ac:12:00:02 brd ff:ff:ff:ff:ff:ff
    inet6 fd00:10::100/64 scope global flags 02
       valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:2/64 scope link
       valid_lft forever preferred_lft forever`,
			expected: "",
		},
	}
	for _, tc := range testCases {
		actual := orca.ParseIPv4Address(tc.ifConfig)
		if actual != tc.expected {
			t.Errorf("FAILED: [%s]. Expected %q, got %q", tc.name, tc.expected, actual)
		}
	}
}
