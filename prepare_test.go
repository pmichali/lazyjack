package orca_test

import (
	"bytes"
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

func TestBuildNodeInfo(t *testing.T) {
	c := &orca.Config{
		Topology: map[string]orca.Node{
			"master": {
				ID: 10,
			},
			"minion": {
				ID: 20,
			},
			"alpha": {
				ID: 30,
			},
		},
		Mgmt: orca.ManagementNetwork{
			Subnet: "fd00:100::",
		},
	}

	ni := orca.BuildNodeInfo(c)
	if len(ni) != 3 {
		t.Errorf("FAILURE: Expected three nodes")
	}
	expected1st := orca.NodeInfo{Name: "alpha", IP: "fd00:100::30", Seen: false}
	expected2nd := orca.NodeInfo{Name: "master", IP: "fd00:100::10", Seen: false}
	expected3rd := orca.NodeInfo{Name: "minion", IP: "fd00:100::20", Seen: false}
	if ni[0] != expected1st {
		t.Errorf("FAILED: First entry does not match. Expected: %+v, got %+v", expected1st, ni[0])
	}
	if ni[1] != expected2nd {
		t.Errorf("FAILED: First entry does not match. Expected: %+v, got %+v", expected2nd, ni[1])
	}
	if ni[2] != expected3rd {
		t.Errorf("FAILED: First entry does not match. Expected: %+v, got %+v", expected3rd, ni[2])
	}
}

func TestMatchingIndexs(t *testing.T) {
	ni := []orca.NodeInfo{
		{
			Name: "master",
		},
		{
			Name: "minionA",
		},
		{
			Name: "minionB",
		},
	}
	idx := orca.MatchingNodeIndex([]byte("10.20.30.40 minionA"), ni)
	if idx != 1 {
		t.Errorf("FAILED: Should have been able to find node 'minionA'")
	}
	idx = orca.MatchingNodeIndex([]byte("10.20.30.40 minionC"), ni)
	if idx != -1 {
		t.Errorf("FAILED: Should not have been able to find node 'minionC'")
	}
}

func TestUpdateEtcHostsContents(t *testing.T) {
	var testCases = []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "Comment existing, add new",
			input: bytes.NewBufferString(`# existing old
10.0.0.2 master
10.0.0.3 minion
`).Bytes(),
			expected: `# existing old
#[X] 10.0.0.2 master
#[X] 10.0.0.3 minion
fd00:20::10 master
fd00:20::20 minion
`,
		},
		{
			name: "Ignore commented, add new",
			input: bytes.NewBufferString(`# ignore commented
10.0.0.2 master
# 10.0.0.3 minion
`).Bytes(),
			expected: `# ignore commented
#[X] 10.0.0.2 master
# 10.0.0.3 minion
fd00:20::10 master
fd00:20::20 minion
`,
		},
		{
			name: "Add new, no existing",
			input: bytes.NewBufferString(`# add new
127.0.0.1 localhost
`).Bytes(),
			expected: `# add new
127.0.0.1 localhost
fd00:20::10 master
fd00:20::20 minion
`,
		},
		{
			name: "Ignore add, already exists",
			input: bytes.NewBufferString(`# ignore existing
10.0.0.2 master
fd00:20::20 minion
`).Bytes(),
			expected: `# ignore existing
#[X] 10.0.0.2 master
fd00:20::20 minion
fd00:20::10 master
`,
		},
		{
			name: "Multiple existing, add new",
			input: bytes.NewBufferString(`# multiple existing
10.0.0.2 master
10.0.0.3 minion
10.0.0.2 master
10.0.0.3 minion
`).Bytes(),
			expected: `# multiple existing
#[X] 10.0.0.2 master
#[X] 10.0.0.3 minion
#[X] 10.0.0.2 master
#[X] 10.0.0.3 minion
fd00:20::10 master
fd00:20::20 minion
`,
		},
	}
	for _, tc := range testCases {
		ni := []orca.NodeInfo{
			{
				Name: "master",
				IP:   "fd00:20::10",
			},
			{
				Name: "minion",
				IP:   "fd00:20::20",
			},
		}

		actual := orca.UpdateHostsInfo(tc.input, ni)
		if string(actual) != tc.expected {
			t.Errorf("FAILED: [%s] mismatch. Expected:\n%s\nActual:\n%s\n", tc.name, tc.expected, string(actual))
		}
	}
}
