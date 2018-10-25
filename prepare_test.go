package lazyjack_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestKubeletDropInContents(t *testing.T) {
	c := &lazyjack.Config{
		Service: lazyjack.ServiceNetwork{
			CIDR: "2001:db8::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "2001:db8::",
			},
		},
	}

	expected := `[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=2001:db8::a --cluster-domain=cluster.local"
`
	actual := lazyjack.CreateKubeletDropInContents(c)
	if actual.String() != expected {
		t.Fatalf("Kubelet drop-in contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func TestKubeletDropInContentsV4(t *testing.T) {
	c := &lazyjack.Config{
		Service: lazyjack.ServiceNetwork{
			CIDR: "10.96.0.0/12",
			Info: lazyjack.NetInfo{
				Mode:   "ipv4",
				Prefix: "10.96.0.",
			},
		},
	}

	expected := `[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=10.96.0.10 --cluster-domain=cluster.local"
`
	actual := lazyjack.CreateKubeletDropInContents(c)
	if actual.String() != expected {
		t.Fatalf("Kubelet drop-in contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func TestCreateKubeletDropInFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		Service: lazyjack.ServiceNetwork{CIDR: "2001:db8::/110"},
		General: lazyjack.GeneralSettings{SystemdArea: basePath},
	}

	err := lazyjack.CreateKubeletDropInFile(c)
	if err != nil {
		t.Fatalf("FAILURE: Expected to be able to create drop-in file: %s", err.Error())
	}
}

func TestFailureToCreateKubeletDropInFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	HelperMakeReadOnly(basePath, t)
	c := &lazyjack.Config{
		Service: lazyjack.ServiceNetwork{CIDR: "2001:db8::/110"},
		General: lazyjack.GeneralSettings{SystemdArea: filepath.Join(basePath, "subdir")},
	}

	err := lazyjack.CreateKubeletDropInFile(c)
	if err == nil {
		t.Fatalf("FAILURE: Expected not to be able to create area for drop-in file")
	}
}

func TestNamedConfContents(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
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
	actual := lazyjack.CreateNamedConfContents(c)
	if actual.String() != expected {
		t.Fatalf("DNS64 named.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func TestNamedConfContentsAllowingAAAA(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
			AllowAAAAUse:   true,
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
    };
};
`
	actual := lazyjack.CreateNamedConfContents(c)
	if actual.String() != expected {
		t.Fatalf("DNS64 named.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func TestCreateSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNotExists: true},
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
	}
	err := lazyjack.CreateSupportNetwork(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create support network: %s", err.Error())
	}
}

func TestFailCreateSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists:     true,
				simCreateNetFail: true,
			},
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
	}
	err := lazyjack.CreateSupportNetwork(c)
	if err == nil {
		t.Fatalf("FAILED: Expected create support network to fail")
	}
	expected := "mock fail create of network"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestSkippingCreateSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{},
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
	}
	err := lazyjack.CreateSupportNetwork(c)
	if err == nil {
		t.Fatalf("FAILED: Expected support network to already exist")
	}
	expected := "skipping - support network already exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestCreateConfigForDNS64(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{mountPoint: volumeMountPoint},
		},
	}

	err := lazyjack.CreateConfigForDNS64(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create DNS64 config file: %s", err.Error())
	}
	conf := filepath.Join(volumeMountPoint, lazyjack.DNS64NamedConf)
	if _, err := os.Stat(conf); os.IsNotExist(err) {
		t.Fatalf("FAILED: Config file %q was not created", conf)
	}
}

func TestFailedDeleteVolumeCreateConfigForDNS64(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteVolumeFail: true,
				mountPoint:          volumeMountPoint,
			},
		},
	}

	err := lazyjack.CreateConfigForDNS64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete existing volume")
	}
	expected := "unable to remove existing volume: mock fail delete of volume"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedCreateVolumeCreateConfigForDNS64(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simCreateVolumeFail: true,
				mountPoint:          volumeMountPoint,
			},
		},
	}

	err := lazyjack.CreateConfigForDNS64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create volume")
	}
	expected := "unable to create volume for DNS64 container use: mock fail create of volume"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedGetVolumeInfoCreateConfigForDNS64(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simInspectVolumeFail: true,
				mountPoint:           volumeMountPoint,
			},
		},
	}

	err := lazyjack.CreateConfigForDNS64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to get volume info")
	}
	expected := "unable to determine mount point for volume: mock fail inspect of volume"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedWriteConfigFileCreateConfigForDNS64(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simBadVolumeInfo: true,
				mountPoint:       volumeMountPoint,
			},
		},
	}

	err := lazyjack.CreateConfigForDNS64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected failure writing config info")
	}
	expected := "unable to create named.conf for DNS64: open /tmp/volume-mount-point-failure/named.conf: no such file or directory"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
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
		actual := lazyjack.ParseIPv4Address(tc.ifConfig)
		if actual != tc.expected {
			t.Fatalf("FAILED: [%s]. Expected %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestBuildNodeInfo(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
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
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}

	ni := lazyjack.BuildNodeInfo(c)
	if len(ni) != 3 {
		t.Fatalf("FAILURE: Expected three nodes")
	}
	expected1st := lazyjack.NodeInfo{Name: "alpha", IP: "fd00:100::30", Seen: false}
	expected2nd := lazyjack.NodeInfo{Name: "master", IP: "fd00:100::10", Seen: false}
	expected3rd := lazyjack.NodeInfo{Name: "minion", IP: "fd00:100::20", Seen: false}
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

func TestBuildNodeInfoUsingSecondMgmtNet(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
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
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
				{
					Prefix: "10.192.0.",
					Mode:   lazyjack.IPv4NetMode,
				},
			},
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Mode: lazyjack.IPv4NetMode,
			},
		},
	}

	ni := lazyjack.BuildNodeInfo(c)
	if len(ni) != 3 {
		t.Fatalf("FAILURE: Expected three nodes")
	}
	expected1st := lazyjack.NodeInfo{Name: "alpha", IP: "10.192.0.30", Seen: false}
	expected2nd := lazyjack.NodeInfo{Name: "master", IP: "10.192.0.10", Seen: false}
	expected3rd := lazyjack.NodeInfo{Name: "minion", IP: "10.192.0.20", Seen: false}
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

func TestAddHostEntries(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
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
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		General: lazyjack.GeneralSettings{
			EtcArea: basePath,
		},
	}

	// Make a file to read
	filename := filepath.Join(basePath, lazyjack.EtcHostsFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create hosts file for test")
	}

	err = lazyjack.AddHostEntries(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to update hosts file: %s", err.Error())
	}
}

func TestMatchingIndexes(t *testing.T) {
	ni := []lazyjack.NodeInfo{
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
	idx := lazyjack.MatchingNodeIndex([]byte("10.20.30.40 minionA"), ni)
	if idx != 1 {
		t.Errorf("FAILED: Should have been able to find node 'minionA'")
	}
	idx = lazyjack.MatchingNodeIndex([]byte("10.20.30.40 minionC"), ni)
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
#[-] 10.0.0.2 master
#[-] 10.0.0.3 minion
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`,
		},
		{
			name: "Ignore commented, add new",
			input: bytes.NewBufferString(`# ignore commented
10.0.0.2 master
# 10.0.0.3 minion
`).Bytes(),
			expected: `# ignore commented
#[-] 10.0.0.2 master
# 10.0.0.3 minion
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`,
		},
		{
			name: "Add new, no existing",
			input: bytes.NewBufferString(`# add new
127.0.0.1 localhost
`).Bytes(),
			expected: `# add new
127.0.0.1 localhost
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`,
		},
		{
			name: "Ignore add, already exists",
			input: bytes.NewBufferString(`# ignore existing
10.0.0.2 master
fd00:20::20 minion
`).Bytes(),
			expected: `# ignore existing
#[-] 10.0.0.2 master
fd00:20::20 minion
fd00:20::10 master  #[+]
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
#[-] 10.0.0.2 master
#[-] 10.0.0.3 minion
#[-] 10.0.0.2 master
#[-] 10.0.0.3 minion
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`,
		},
		{
			name: "relace previous",
			input: bytes.NewBufferString(`# replace previous
10.0.0.2 master
fd00:bad::99 minion  #[+]
`).Bytes(),
			expected: `# replace previous
#[-] 10.0.0.2 master
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`,
		},
	}
	for _, tc := range testCases {
		ni := []lazyjack.NodeInfo{
			{
				Name: "master",
				IP:   "fd00:20::10",
			},
			{
				Name: "minion",
				IP:   "fd00:20::20",
			},
		}

		actual := lazyjack.UpdateHostsInfo(tc.input, ni)
		if string(actual) != tc.expected {
			t.Errorf("FAILED: [%s] mismatch. Expected:\n%s\nActual:\n%s\n", tc.name, tc.expected, string(actual))
		}
	}
}

func TestCalcNameServerV6(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{ServerIP: "2001:db8::100"},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   4,
	}

	ns := lazyjack.CalcNameServer(n, c)
	expected := "2001:db8::100"
	if ns != expected {
		t.Errorf("Expected nameserver %q, got %q", expected, ns)
	}
}

func TestCalcNameServerNodeIP(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv4NetMode,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "10.192.0.",
					Mode:   lazyjack.IPv4NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			Info: lazyjack.NetInfo{
				Mode: lazyjack.IPv4NetMode,
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   4,
	}

	ns := lazyjack.CalcNameServer(n, c)
	expected := "10.192.0.4"
	if ns != expected {
		t.Errorf("Expected nameserver %q, got %q", expected, ns)
	}
}

func TestCalcNameServerUsingSecondNodeIP(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "10.192.0.",
					Mode:   lazyjack.IPv4NetMode,
				},
				{
					Prefix: "fd00:20::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			Info: lazyjack.NetInfo{
				Mode: lazyjack.IPv6NetMode,
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   5,
	}

	ns := lazyjack.CalcNameServer(n, c)
	expected := "fd00:20::5"
	if ns != expected {
		t.Errorf("Expected nameserver %q, got %q", expected, ns)
	}
}

func TestUpdateResolvConfContents(t *testing.T) {
	var testCases = []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "no nameservers",
			input: bytes.NewBufferString(`# no nameservers
search example.com
`).Bytes(),
			expected: `# no nameservers
search example.com
nameserver fd00:10::100  #[+]
`,
		},
		{
			name: "prepend to existing",
			input: bytes.NewBufferString(`# prepend to existing
search example.com
nameserver 8.8.8.8
nameserver 8.8.4.4
`).Bytes(),
			expected: `# prepend to existing
search example.com
nameserver fd00:10::100  #[+]
nameserver 8.8.8.8
nameserver 8.8.4.4
`,
		},
		{
			name: "not first entry",
			input: bytes.NewBufferString(`# not first entry
search example.com
nameserver 8.8.8.8
nameserver fd00:10::100
nameserver 8.8.4.4
`).Bytes(),
			expected: `# not first entry
search example.com
nameserver fd00:10::100  #[+]
nameserver 8.8.8.8
#[-] nameserver fd00:10::100
nameserver 8.8.4.4
`,
		},
		{
			name: "already have",
			input: bytes.NewBufferString(`# already have
search example.com
nameserver fd00:10::100
nameserver 8.8.8.8
`).Bytes(),
			expected: `# already have
search example.com
nameserver fd00:10::100
nameserver 8.8.8.8
`,
		},
		{
			name: "changed value",
			input: bytes.NewBufferString(`# changed value
search example.com
nameserver fd00:10::999  #[+]
nameserver 8.8.8.8
`).Bytes(),
			expected: `# changed value
search example.com
nameserver fd00:10::100  #[+]
nameserver 8.8.8.8
`,
		},
	}

	for _, tc := range testCases {
		actual := lazyjack.UpdateResolvConfInfo(tc.input, "fd00:10::100")
		if string(actual) != tc.expected {
			t.Errorf("FAILED: [%s] mismatch.\nExpected:\n%s\nActual:\n%s\n", tc.name, tc.expected, string(actual))
		}
	}

}

func TestAddResolvConfEntry(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{ServerIP: "2001:db8::100"},
		General: lazyjack.GeneralSettings{
			Mode:    lazyjack.IPv6NetMode,
			EtcArea: basePath,
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   4,
	}

	// Make a file to read
	filename := filepath.Join(basePath, lazyjack.EtcResolvConfFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	err = lazyjack.AddResolvConfEntry(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to update resolv.conf file: %s", err.Error())
	}
}

func TestFindHostIPForNAT64(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: false,
			},
			"minion1": {
				ID:            0x20,
				IsNAT64Server: true,
			},
			"minion2": {
				ID:            0x30,
				IsNAT64Server: false,
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	gw, ok := lazyjack.FindHostIPForNAT64(c)
	if !ok {
		t.Fatalf("Expected to find node with NAT64 server")
	}
	if gw != "fd00:100::20" {
		t.Fatalf("Incorrect GW IP from node with NAT64 server")
	}
	bad := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: false,
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	gw, ok = lazyjack.FindHostIPForNAT64(bad)
	if ok {
		t.Fatalf("Expected no NAT64 server to be found")
	}
}

func TestCollectKubeAdmConfigInfo(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:      "64rxu8.yvrzfofegfmyy1no",
			K8sVersion: "1.12",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}
	actual := lazyjack.CollectKubeAdmConfigInfo(n, c)
	var expected string
	expected = "fd00:100::10"
	if actual.AdvertiseAddress != expected {
		t.Errorf("Expected advertise address %q, got %q", expected, actual.AdvertiseAddress)
	}
	expected = "64rxu8.yvrzfofegfmyy1no"
	if actual.AuthToken != expected {
		t.Errorf("Expected auth token %q, got %q", expected, actual.AuthToken)
	}
	expected = "::"
	if actual.BindAddress != expected {
		t.Errorf("Expected bind address %q, got %q", expected, actual.BindAddress)
	}
	expected = "fd00:30::a"
	if actual.DNS_ServiceIP != expected {
		t.Errorf("Expected DNS IP %q, got %q", expected, actual.DNS_ServiceIP)
	}
	expected = "fd00:40::/72"
	if actual.PodNetworkCIDR != expected {
		t.Errorf("Expected pod CIDR %q, got %q", expected, actual.PodNetworkCIDR)
	}
	expected = "kubernetesVersion: \"1.12\""
	if actual.K8sVersion != expected {
		t.Errorf("Expected Kubernetes version %q, got %q", expected, actual.K8sVersion)
	}
}

func TestCollectKubeAdmConfigInfo2(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Insecure: true,
		},
		Pod: lazyjack.PodNetwork{
			CIDR2: "10.244.0.0/16",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
				{
					Mode: lazyjack.IPv4NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "10.96.0.12",
			Info: lazyjack.NetInfo{
				Mode:   lazyjack.IPv4NetMode,
				Prefix: "10.96.0.",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
				{
					Prefix: "10.192.0.",
					Mode:   lazyjack.IPv4NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}
	actual := lazyjack.CollectKubeAdmConfigInfo(n, c)
	var expected string
	expected = "10.192.0.10"
	if actual.AdvertiseAddress != expected {
		t.Errorf("Expected advertise address %q, got %q", expected, actual.AdvertiseAddress)
	}
	expected = "abcdef.abcdefghijklmnop"
	if actual.AuthToken != expected {
		t.Errorf("Expected auth token %q, got %q", expected, actual.AuthToken)
	}
	expected = "0.0.0.0"
	if actual.BindAddress != expected {
		t.Errorf("Expected bind address %q, got %q", expected, actual.BindAddress)
	}
	expected = "10.96.0.10"
	if actual.DNS_ServiceIP != expected {
		t.Errorf("Expected DNS IP %q, got %q", expected, actual.DNS_ServiceIP)
	}
	expected = "10.244.0.0/16"
	if actual.PodNetworkCIDR != expected {
		t.Errorf("Expected pod CIDR %q, got %q", expected, actual.PodNetworkCIDR)
	}
	expected = "# kubernetesVersion:"
	if actual.K8sVersion != expected {
		t.Errorf("Expected Kubernetes version %q, got %q", expected, actual.K8sVersion)
	}
}

func TestKubeAdmConfigContents_1_10_V6(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.10",
			K8sVersion:     "v1.10.3",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   lazyjack.IPv6NetMode,
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.10 (and older) based config
api:
  advertiseAddress: "fd00:100::10"
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha1
featureGates: {CoreDNS: false}
kind: MasterConfiguration
kubernetesVersion: "v1.10.3"
networking:
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
token: "56cdce.7b18ad347f3de81c"
tokenTTL: 0s
nodeName: my-master
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContents_1_10_V4(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.10",
			K8sVersion:     "v1.10.3",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "10.96.0.0/12",
			Info: lazyjack.NetInfo{
				Mode:   "ipv4",
				Prefix: "10.96.0.",
			},
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "10.244.0.0/16",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv4NetMode,
				},
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "10.192.0.",
					Mode:   lazyjack.IPv4NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   4,
	}

	expected := `# V1.10 (and older) based config
api:
  advertiseAddress: "10.192.0.4"
apiServerExtraArgs:
  insecure-bind-address: "0.0.0.0"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha1
featureGates: {CoreDNS: false}
kind: MasterConfiguration
kubernetesVersion: "v1.10.3"
networking:
  # podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
token: "56cdce.7b18ad347f3de81c"
tokenTTL: 0s
nodeName: my-master
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForKubeAdm_1_11(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.11",
			K8sVersion:     "v1.11.1",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.11 based config
api:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
  controlPlaneEndpoint: ""
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha2
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 56cdce.7b18ad347f3de81c
  ttl: 0s
  usages:
  - signing
  - authentication
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
kind: MasterConfiguration
kubeProxy:
  config:
    bindAddress: "::"
    clientConnection:
      acceptContentTypes: ""
      burst: 10
      contentType: application/vnd.kubernetes.protobuf
      kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
      qps: 5
    # clusterCIDR: ""
    configSyncPeriod: 15m0s
    # conntrack:
    #   max: null
    #   maxPerCore: 32768
    #   min: 131072
    #   tcpCloseWaitTimeout: 1h0m0s
    #   tcpEstablishedTimeout: 24h0m0s
    enableProfiling: false
    healthzBindAddress: 0.0.0.0:10256
    hostnameOverride: ""
    iptables:
      masqueradeAll: false
      masqueradeBit: 14
      minSyncPeriod: 0s
      syncPeriod: 30s
    ipvs:
      excludeCIDRs: null
      minSyncPeriod: 0s
      scheduler: ""
      syncPeriod: 30s
    metricsBindAddress: 127.0.0.1:10249
    mode: ""
    nodePortAddresses: null
    oomScoreAdj: -999
    portRange: ""
    resourceContainer: /kube-proxy
    udpIdleTimeout: 250ms
kubeletConfiguration:
  baseConfig:
    address: 0.0.0.0
    authentication:
      anonymous:
        enabled: false
      webhook:
        cacheTTL: 2m0s
        enabled: true
      x509:
        clientCAFile: /etc/kubernetes/pki/ca.crt
    authorization:
      mode: Webhook
      webhook:
        cacheAuthorizedTTL: 5m0s
        cacheUnauthorizedTTL: 30s
    cgroupDriver: cgroupfs
    cgroupsPerQOS: true
    clusterDNS:
    - "fd00:30::a"
    clusterDomain: cluster.local
    containerLogMaxFiles: 5
    containerLogMaxSize: 10Mi
    contentType: application/vnd.kubernetes.protobuf
    cpuCFSQuota: true
    cpuManagerPolicy: none
    cpuManagerReconcilePeriod: 10s
    enableControllerAttachDetach: true
    enableDebuggingHandlers: true
    enforceNodeAllocatable:
    - pods
    eventBurst: 10
    eventRecordQPS: 5
    evictionHard:
      imagefs.available: 15%
      memory.available: 100Mi
      nodefs.available: 10%
      nodefs.inodesFree: 5%
    evictionPressureTransitionPeriod: 5m0s
    failSwapOn: true
    fileCheckFrequency: 20s
    hairpinMode: promiscuous-bridge
    healthzBindAddress: 127.0.0.1
    healthzPort: 10248
    httpCheckFrequency: 20s
    imageGCHighThresholdPercent: 85
    imageGCLowThresholdPercent: 80
    imageMinimumGCAge: 2m0s
    iptablesDropBit: 15
    iptablesMasqueradeBit: 14
    kubeAPIBurst: 10
    kubeAPIQPS: 5
    makeIPTablesUtilChains: true
    maxOpenFiles: 1000000
    maxPods: 110
    nodeStatusUpdateFrequency: 10s
    oomScoreAdj: -999
    podPidsLimit: -1
    # port: 10250
    registryBurst: 10
    registryPullQPS: 5
    resolvConf: /etc/resolv.conf
    rotateCertificates: true
    runtimeRequestTimeout: 2m0s
    serializeImagePulls: true
    staticPodPath: /etc/kubernetes/manifests
    streamingConnectionIdleTimeout: 4h0m0s
    syncFrequency: 1m0s
    volumeStatsAggPeriod: 1m0s
kubernetesVersion: "v1.11.1"
networking:
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
nodeRegistration:
  name: my-master
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForKubeAdm_1_11_V4(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.11",
			K8sVersion:     "v1.11.1",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "10.244.0.0/24",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv4NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "10.96.0.0/12",
			Info: lazyjack.NetInfo{
				Mode:   "ipv4",
				Prefix: "10.96.0.",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "10.192.0.",
					Mode:   lazyjack.IPv4NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.11 based config
api:
  advertiseAddress: "10.192.0.10"
  bindPort: 6443
  controlPlaneEndpoint: ""
apiServerExtraArgs:
  insecure-bind-address: "0.0.0.0"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha2
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 56cdce.7b18ad347f3de81c
  ttl: 0s
  usages:
  - signing
  - authentication
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
kind: MasterConfiguration
kubeProxy:
  config:
    bindAddress: "0.0.0.0"
    clientConnection:
      acceptContentTypes: ""
      burst: 10
      contentType: application/vnd.kubernetes.protobuf
      kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
      qps: 5
    # clusterCIDR: ""
    configSyncPeriod: 15m0s
    # conntrack:
    #   max: null
    #   maxPerCore: 32768
    #   min: 131072
    #   tcpCloseWaitTimeout: 1h0m0s
    #   tcpEstablishedTimeout: 24h0m0s
    enableProfiling: false
    healthzBindAddress: 0.0.0.0:10256
    hostnameOverride: ""
    iptables:
      masqueradeAll: false
      masqueradeBit: 14
      minSyncPeriod: 0s
      syncPeriod: 30s
    ipvs:
      excludeCIDRs: null
      minSyncPeriod: 0s
      scheduler: ""
      syncPeriod: 30s
    metricsBindAddress: 127.0.0.1:10249
    mode: ""
    nodePortAddresses: null
    oomScoreAdj: -999
    portRange: ""
    resourceContainer: /kube-proxy
    udpIdleTimeout: 250ms
kubeletConfiguration:
  baseConfig:
    address: 0.0.0.0
    authentication:
      anonymous:
        enabled: false
      webhook:
        cacheTTL: 2m0s
        enabled: true
      x509:
        clientCAFile: /etc/kubernetes/pki/ca.crt
    authorization:
      mode: Webhook
      webhook:
        cacheAuthorizedTTL: 5m0s
        cacheUnauthorizedTTL: 30s
    cgroupDriver: cgroupfs
    cgroupsPerQOS: true
    clusterDNS:
    - "10.96.0.10"
    clusterDomain: cluster.local
    containerLogMaxFiles: 5
    containerLogMaxSize: 10Mi
    contentType: application/vnd.kubernetes.protobuf
    cpuCFSQuota: true
    cpuManagerPolicy: none
    cpuManagerReconcilePeriod: 10s
    enableControllerAttachDetach: true
    enableDebuggingHandlers: true
    enforceNodeAllocatable:
    - pods
    eventBurst: 10
    eventRecordQPS: 5
    evictionHard:
      imagefs.available: 15%
      memory.available: 100Mi
      nodefs.available: 10%
      nodefs.inodesFree: 5%
    evictionPressureTransitionPeriod: 5m0s
    failSwapOn: true
    fileCheckFrequency: 20s
    hairpinMode: promiscuous-bridge
    healthzBindAddress: 127.0.0.1
    healthzPort: 10248
    httpCheckFrequency: 20s
    imageGCHighThresholdPercent: 85
    imageGCLowThresholdPercent: 80
    imageMinimumGCAge: 2m0s
    iptablesDropBit: 15
    iptablesMasqueradeBit: 14
    kubeAPIBurst: 10
    kubeAPIQPS: 5
    makeIPTablesUtilChains: true
    maxOpenFiles: 1000000
    maxPods: 110
    nodeStatusUpdateFrequency: 10s
    oomScoreAdj: -999
    podPidsLimit: -1
    # port: 10250
    registryBurst: 10
    registryPullQPS: 5
    resolvConf: /etc/resolv.conf
    rotateCertificates: true
    runtimeRequestTimeout: 2m0s
    serializeImagePulls: true
    staticPodPath: /etc/kubernetes/manifests
    streamingConnectionIdleTimeout: 4h0m0s
    syncFrequency: 1m0s
    volumeStatsAggPeriod: 1m0s
kubernetesVersion: "v1.11.1"
networking:
  # podSubnet: "10.244.0.0/24"
  serviceSubnet: "10.96.0.0/12"
nodeRegistration:
  name: my-master
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForKubeAdm_1_12(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "64rxu8.yvrzfofegfmyy1no",
			KubeAdmVersion: "1.12",
			K8sVersion:     "v1.12.0",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.12 based config
apiEndpoint:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1alpha3
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 64rxu8.yvrzfofegfmyy1no
  ttl: 0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: my-master
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha3
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
kubernetesVersion: "v1.12.0"
networking:
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForKubeAdm_1_12_no_k8s_version(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "64rxu8.yvrzfofegfmyy1no",
			KubeAdmVersion: "1.12",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.12 based config
apiEndpoint:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1alpha3
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 64rxu8.yvrzfofegfmyy1no
  ttl: 0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: my-master
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha3
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
# kubernetesVersion:
networking:
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForKubeAdm_1_13(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.13",
			K8sVersion:     "v1.13.0",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.13 based config
apiEndpoint:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1beta1
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 56cdce.7b18ad347f3de81c
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: my-master
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1beta1
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
kubernetesVersion: "v1.13.0"
networking:
  dnsDomain: cluster.local
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
unifiedControlPlaneImage: ""
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
bindAddress: "::"
clientConnection:
  acceptContentTypes: ""
  burst: 10
  contentType: application/vnd.kubernetes.protobuf
  kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
  qps: 5
# clusterCIDR: ""
configSyncPeriod: 15m0s
# conntrack:
#   max: null
#   maxPerCore: 32768
#   min: 131072
#   tcpCloseWaitTimeout: 1h0m0s
#   tcpEstablishedTimeout: 24h0m0s
enableProfiling: false
healthzBindAddress: 0.0.0.0:10256
hostnameOverride: ""
iptables:
  masqueradeAll: false
  masqueradeBit: 14
  minSyncPeriod: 0s
  syncPeriod: 30s
ipvs:
  excludeCIDRs: null
  minSyncPeriod: 0s
  scheduler: ""
  syncPeriod: 30s
kind: KubeProxyConfiguration
metricsBindAddress: 127.0.0.1:10249
mode: ""
nodePortAddresses: null
oomScoreAdj: -999
portRange: ""
resourceContainer: /kube-proxy
udpIdleTimeout: 250ms
---
address: 0.0.0.0
apiVersion: kubelet.config.k8s.io/v1beta1
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver: cgroupfs
cgroupsPerQOS: true
clusterDNS:
- "fd00:30::a"
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
enforceNodeAllocatable:
- pods
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kind: KubeletConfiguration
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
# port: 10250
registryBurst: 10
registryPullQPS: 5
resolvConf: /etc/resolv.conf
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: true
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForKubeAdm_1_13_latest_k8s(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.13",
			K8sVersion:     "latest",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.13 based config
apiEndpoint:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1beta1
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: 56cdce.7b18ad347f3de81c
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: my-master
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1beta1
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
kubernetesVersion: "latest"
networking:
  dnsDomain: cluster.local
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
unifiedControlPlaneImage: ""
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
bindAddress: "::"
clientConnection:
  acceptContentTypes: ""
  burst: 10
  contentType: application/vnd.kubernetes.protobuf
  kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
  qps: 5
# clusterCIDR: ""
configSyncPeriod: 15m0s
# conntrack:
#   max: null
#   maxPerCore: 32768
#   min: 131072
#   tcpCloseWaitTimeout: 1h0m0s
#   tcpEstablishedTimeout: 24h0m0s
enableProfiling: false
healthzBindAddress: 0.0.0.0:10256
hostnameOverride: ""
iptables:
  masqueradeAll: false
  masqueradeBit: 14
  minSyncPeriod: 0s
  syncPeriod: 30s
ipvs:
  excludeCIDRs: null
  minSyncPeriod: 0s
  scheduler: ""
  syncPeriod: 30s
kind: KubeProxyConfiguration
metricsBindAddress: 127.0.0.1:10249
mode: ""
nodePortAddresses: null
oomScoreAdj: -999
portRange: ""
resourceContainer: /kube-proxy
udpIdleTimeout: 250ms
---
address: 0.0.0.0
apiVersion: kubelet.config.k8s.io/v1beta1
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver: cgroupfs
cgroupsPerQOS: true
clusterDNS:
- "fd00:30::a"
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
enforceNodeAllocatable:
- pods
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kind: KubeletConfiguration
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
# port: 10250
registryBurst: 10
registryPullQPS: 5
resolvConf: /etc/resolv.conf
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: true
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForInsecureKubeAdm_1_11(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Insecure:       true,
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.11",
			K8sVersion:     "v1.11.1",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.11 based config
api:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
  controlPlaneEndpoint: ""
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha2
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.abcdefghijklmnop
  ttl: 0s
  usages:
  - signing
  - authentication
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
kind: MasterConfiguration
kubeProxy:
  config:
    bindAddress: "::"
    clientConnection:
      acceptContentTypes: ""
      burst: 10
      contentType: application/vnd.kubernetes.protobuf
      kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
      qps: 5
    # clusterCIDR: ""
    configSyncPeriod: 15m0s
    # conntrack:
    #   max: null
    #   maxPerCore: 32768
    #   min: 131072
    #   tcpCloseWaitTimeout: 1h0m0s
    #   tcpEstablishedTimeout: 24h0m0s
    enableProfiling: false
    healthzBindAddress: 0.0.0.0:10256
    hostnameOverride: ""
    iptables:
      masqueradeAll: false
      masqueradeBit: 14
      minSyncPeriod: 0s
      syncPeriod: 30s
    ipvs:
      excludeCIDRs: null
      minSyncPeriod: 0s
      scheduler: ""
      syncPeriod: 30s
    metricsBindAddress: 127.0.0.1:10249
    mode: ""
    nodePortAddresses: null
    oomScoreAdj: -999
    portRange: ""
    resourceContainer: /kube-proxy
    udpIdleTimeout: 250ms
kubeletConfiguration:
  baseConfig:
    address: 0.0.0.0
    authentication:
      anonymous:
        enabled: false
      webhook:
        cacheTTL: 2m0s
        enabled: true
      x509:
        clientCAFile: /etc/kubernetes/pki/ca.crt
    authorization:
      mode: Webhook
      webhook:
        cacheAuthorizedTTL: 5m0s
        cacheUnauthorizedTTL: 30s
    cgroupDriver: cgroupfs
    cgroupsPerQOS: true
    clusterDNS:
    - "fd00:30::a"
    clusterDomain: cluster.local
    containerLogMaxFiles: 5
    containerLogMaxSize: 10Mi
    contentType: application/vnd.kubernetes.protobuf
    cpuCFSQuota: true
    cpuManagerPolicy: none
    cpuManagerReconcilePeriod: 10s
    enableControllerAttachDetach: true
    enableDebuggingHandlers: true
    enforceNodeAllocatable:
    - pods
    eventBurst: 10
    eventRecordQPS: 5
    evictionHard:
      imagefs.available: 15%
      memory.available: 100Mi
      nodefs.available: 10%
      nodefs.inodesFree: 5%
    evictionPressureTransitionPeriod: 5m0s
    failSwapOn: true
    fileCheckFrequency: 20s
    hairpinMode: promiscuous-bridge
    healthzBindAddress: 127.0.0.1
    healthzPort: 10248
    httpCheckFrequency: 20s
    imageGCHighThresholdPercent: 85
    imageGCLowThresholdPercent: 80
    imageMinimumGCAge: 2m0s
    iptablesDropBit: 15
    iptablesMasqueradeBit: 14
    kubeAPIBurst: 10
    kubeAPIQPS: 5
    makeIPTablesUtilChains: true
    maxOpenFiles: 1000000
    maxPods: 110
    nodeStatusUpdateFrequency: 10s
    oomScoreAdj: -999
    podPidsLimit: -1
    # port: 10250
    registryBurst: 10
    registryPullQPS: 5
    resolvConf: /etc/resolv.conf
    rotateCertificates: true
    runtimeRequestTimeout: 2m0s
    serializeImagePulls: true
    staticPodPath: /etc/kubernetes/manifests
    streamingConnectionIdleTimeout: 4h0m0s
    syncFrequency: 1m0s
    volumeStatsAggPeriod: 1m0s
kubernetesVersion: "v1.11.1"
networking:
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
nodeRegistration:
  name: my-master
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForInsecureKubeAdm_1_12(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Insecure:       true,
			Token:          "64rxu8.yvrzfofegfmyy1no",
			KubeAdmVersion: "1.12",
			K8sVersion:     "v1.12.0",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.12 based config
apiEndpoint:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1alpha3
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.abcdefghijklmnop
  ttl: 0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: my-master
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1alpha3
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
kubernetesVersion: "v1.12.0"
networking:
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
unifiedControlPlaneImage: ""
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestKubeAdmConfigContentsForInsecureKubeAdm_1_13(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Insecure:       true,
			Token:          "56cdce.7b18ad347f3de81c",
			KubeAdmVersion: "1.13",
			K8sVersion:     "v1.13.0",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
			Info: [2]lazyjack.NetInfo{
				{
					Mode: lazyjack.IPv6NetMode,
				},
			},
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
			Info: lazyjack.NetInfo{
				Mode:   "ipv6",
				Prefix: "fd00:30::",
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	expected := `# V1.13 based config
apiEndpoint:
  advertiseAddress: "fd00:100::10"
  bindPort: 6443
apiVersion: kubeadm.k8s.io/v1beta1
bootstrapTokens:
- groups:
  - system:bootstrappers:kubeadm:default-node-token
  token: abcdef.abcdefghijklmnop
  ttl: 24h0m0s
  usages:
  - signing
  - authentication
kind: InitConfiguration
nodeRegistration:
  criSocket: /var/run/dockershim.sock
  name: my-master
  taints:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
---
apiServerExtraArgs:
  insecure-bind-address: "::"
  insecure-port: "8080"
apiVersion: kubeadm.k8s.io/v1beta1
auditPolicy:
  logDir: /var/log/kubernetes/audit
  logMaxAge: 2
  path: ""
certificatesDir: /etc/kubernetes/pki
# clusterName: kubernetes
controlPlaneEndpoint: ""
etcd:
  local:
    dataDir: /var/lib/etcd
    image: ""
featureGates: {CoreDNS: false}
imageRepository: k8s.gcr.io
kind: ClusterConfiguration
kubernetesVersion: "v1.13.0"
networking:
  dnsDomain: cluster.local
  # podSubnet: "fd00:40::/72"
  serviceSubnet: "fd00:30::/110"
unifiedControlPlaneImage: ""
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
bindAddress: "::"
clientConnection:
  acceptContentTypes: ""
  burst: 10
  contentType: application/vnd.kubernetes.protobuf
  kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
  qps: 5
# clusterCIDR: ""
configSyncPeriod: 15m0s
# conntrack:
#   max: null
#   maxPerCore: 32768
#   min: 131072
#   tcpCloseWaitTimeout: 1h0m0s
#   tcpEstablishedTimeout: 24h0m0s
enableProfiling: false
healthzBindAddress: 0.0.0.0:10256
hostnameOverride: ""
iptables:
  masqueradeAll: false
  masqueradeBit: 14
  minSyncPeriod: 0s
  syncPeriod: 30s
ipvs:
  excludeCIDRs: null
  minSyncPeriod: 0s
  scheduler: ""
  syncPeriod: 30s
kind: KubeProxyConfiguration
metricsBindAddress: 127.0.0.1:10249
mode: ""
nodePortAddresses: null
oomScoreAdj: -999
portRange: ""
resourceContainer: /kube-proxy
udpIdleTimeout: 250ms
---
address: 0.0.0.0
apiVersion: kubelet.config.k8s.io/v1beta1
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver: cgroupfs
cgroupsPerQOS: true
clusterDNS:
- "fd00:30::a"
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
enforceNodeAllocatable:
- pods
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kind: KubeletConfiguration
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
# port: 10250
registryBurst: 10
registryPullQPS: 5
resolvConf: /etc/resolv.conf
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: true
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s
`
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Fatalf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
	}
}

func TestCreateKubeAdmConfFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:    "56cdce.7b18ad347f3de81c",
			WorkArea: basePath,
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	err := lazyjack.CreateKubeAdmConfigFile(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create KubeAdm config file: %s", err.Error())
	}
}

func TestFailingCreateKubeAdmConfFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:    "56cdce.7b18ad347f3de81c",
			WorkArea: basePath,
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	HelperMakeReadOnly(basePath, t)

	err := lazyjack.CreateKubeAdmConfigFile(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create KubeAdm config file")
	}
}

func TestCreateRouteToNAT64ServerForDNS64SubnetForNATServer(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		General: lazyjack.GeneralSettings{NetMgr: nm},
	}
	masterNode := &lazyjack.Node{
		Name:          "master",
		ID:            0x10,
		IsNAT64Server: true,
		IsDNS64Server: true,
	}
	err := lazyjack.CreateRouteToNAT64ServerForDNS64Subnet(masterNode, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create route: %s", err.Error())
	}
}

func TestCreateRouteToNAT64ServerForDNS64SubnetForNonNATServer(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
		},
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	minionNode := &lazyjack.Node{
		Name:          "minion1",
		ID:            20,
		Interface:     "eth2",
		IsNAT64Server: false,
		IsDNS64Server: false,
	}
	err := lazyjack.CreateRouteToNAT64ServerForDNS64Subnet(minionNode, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create route: %s", err.Error())
	}
}

func TestFailedNoNATServerCreateRouteToNAT64ServerForDNS64Subnet(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
		},
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	minionNode := &lazyjack.Node{
		Name:          "minion1",
		ID:            20,
		Interface:     "eth2",
		IsNAT64Server: false,
		IsDNS64Server: false,
	}
	err := lazyjack.CreateRouteToNAT64ServerForDNS64Subnet(minionNode, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find NAT server: %s", err.Error())
	}
	expected := "unable to find node with NAT64 server configured"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRouteAddCreateRouteToNAT64ServerForDNS64SubnetForNATServer(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		General: lazyjack.GeneralSettings{NetMgr: nm},
	}
	masterNode := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		IsNAT64Server: true,
		IsDNS64Server: true,
	}
	err := lazyjack.CreateRouteToNAT64ServerForDNS64Subnet(masterNode, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestRouteExistsCreateRouteToNAT64ServerForDNS64SubnetForNATServer(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteExists: true}}
	c := &lazyjack.Config{
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		General: lazyjack.GeneralSettings{NetMgr: nm},
	}
	masterNode := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		IsNAT64Server: true,
		IsDNS64Server: true,
	}
	err := lazyjack.CreateRouteToNAT64ServerForDNS64Subnet(masterNode, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create route: %s", err.Error())
	}
}

func TestSkipNAT64ServerForCreateRouteToSupportNetworkForOtherNodes(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		IsNAT64Server: true,
		IsDNS64Server: true,
	}
	err := lazyjack.CreateRouteToSupportNetworkForOtherNodes(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected no actions for NAT64/DNS64 server")
	}
}

func TestCreateRouteToSupportNetworkForOtherNodes(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{CIDR: "fd00:10::/64"},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		Interface:     "eth1",
		IsNAT64Server: false,
		IsDNS64Server: false,
	}
	err := lazyjack.CreateRouteToSupportNetworkForOtherNodes(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to add route: %s", err.Error())
	}
}

func TestFailedRouteAddCreateRouteToSupportNetworkForOtherNodes(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{CIDR: "fd00:10::/64"},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		Interface:     "eth1",
		IsNAT64Server: false,
		IsDNS64Server: false,
	}
	err := lazyjack.CreateRouteToSupportNetworkForOtherNodes(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to add route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRouteExistsCreateRouteToSupportNetworkForOtherNodes(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteExists: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{CIDR: "fd00:10::/64"},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		Interface:     "eth1",
		IsNAT64Server: false,
		IsDNS64Server: false,
	}
	err := lazyjack.CreateRouteToSupportNetworkForOtherNodes(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected existing route to pass: %s", err.Error())
	}
}

func TestFailedNoNatServerCreateRouteToSupportNetworkForOtherNodes(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
		},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{CIDR: "fd00:10::/64"},
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Size:   64,
				},
			},
		},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		Interface:     "eth1",
		IsNAT64Server: false,
		IsDNS64Server: false,
	}
	err := lazyjack.CreateRouteToSupportNetworkForOtherNodes(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to find NAT64/DNS64 server node")
	}
	expected := "unable to find node with NAT64 server configured"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestConfigureManagementInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID: 10,
			},
		},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Size:   64,
				},
			},
		},
		Pod: lazyjack.PodNetwork{
			MTU: 9000,
		},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:      "master",
		ID:        10,
		Interface: "eth1",
	}
	err := lazyjack.ConfigureManagementInterface(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to configure interface: %s", err.Error())
	}

}

func TestFailedAddAddressConfigureManagementInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simReplaceFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID: 10,
			},
		},
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
					Mode:   lazyjack.IPv6NetMode,
					Size:   64,
				},
			},
		},
		Pod: lazyjack.PodNetwork{
			MTU: 9000,
		},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:      "master",
		ID:        10,
		Interface: "eth1",
	}
	err := lazyjack.ConfigureManagementInterface(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to configure interface")
	}
	expected := "unable to add ip \"fd00:100::a/64\" to interface \"eth1\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestPrepareClusterNode(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	systemdArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemdArea, t)
	defer HelperCleanupArea(systemdArea, t)

	// Make needed files
	filename := filepath.Join(etcArea, lazyjack.EtcHostsFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create hosts file for test")
	}
	filename = filepath.Join(etcArea, lazyjack.EtcResolvConfFile)
	err = ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
				IsDNS64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr:      nm,
			WorkArea:    workArea,
			EtcArea:     etcArea,
			SystemdArea: systemdArea,
			Mode:        lazyjack.IPv6NetMode,
		},
		Support: lazyjack.SupportNetwork{
			CIDR:   "fd00:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		DNS64: lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
	}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:          "master",
		ID:            10,
		Interface:     "eth1",
		IsNAT64Server: false,
		IsDNS64Server: false,
		IsMaster:      true,
	}

	err = lazyjack.PrepareClusterNode(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to prepare cluster node: %s", err.Error())
	}
}

func TestNotExistsEnsureDNS64Server(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists: true,
				mountPoint:   volumeMountPoint,
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to run DNS64 container: %v", err)
	}
}

func TestExistsButNotRunningEnsureDNS64Server(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{mountPoint: volumeMountPoint},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to run DNS64 container: %v", err)
	}
}

func TestSkipRunningEnsureDNS64Server(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simRunning: true,
				mountPoint: volumeMountPoint,
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected DNS64 container to already be running")
	}
	expected := "skipping - DNS64 container (bind9) already running"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRemoveOldEnsureDNS64Server(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteContainerFail: true,
				mountPoint:             volumeMountPoint,
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail deleting old DNS64 container")
	}
	expected := "unable to remove existing (non-running) DNS64 container: mock fail delete of container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedConfigCreateEnsureDNS64Server(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simCreateVolumeFail: true,
				mountPoint:          volumeMountPoint,
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail to create config for DNS64")
	}
	expected := "unable to create volume for DNS64 container use: mock fail create of volume"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRunEnsureDNS64Server(t *testing.T) {
	volumeMountPoint := TempFileName(os.TempDir(), "-dns64")
	HelperSetupArea(volumeMountPoint, t)
	defer HelperCleanupArea(volumeMountPoint, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simRunFailed: true,
				mountPoint:   volumeMountPoint,
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail running DNS64 container")
	}
	expected := "mock fail to run container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestRemoveIPv4AddressOnDNS64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{},
		},
	}
	err := lazyjack.RemoveIPv4AddressOnDNS64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove IPv4 address: %v", err)
	}
}

func TestFailedFindAddressRemoveIPv4AddressOnDNS64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNoV4Interface: true},
		},
	}
	err := lazyjack.RemoveIPv4AddressOnDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail finding IPv4 address")
	}
	expected := "unable to find IPv4 address on eth0 of DNS64 container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedGetConfigRemoveIPv4AddressOnDNS64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simInterfaceGetFail: true},
		},
	}
	err := lazyjack.RemoveIPv4AddressOnDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail removing IPv4 address")
	}
	expected := "mock fail getting interface info"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRemoveIPv4AddressOnDNS64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simDeleteInterfaceFail: true},
		},
	}
	err := lazyjack.RemoveIPv4AddressOnDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail removing IPv4 address")
	}
	expected := "mock fail delete of IP"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestAddRouteForDNS64Network(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{},
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		DNS64: lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
	}
	err := lazyjack.AddRouteForDNS64Network(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to add V6 route: %v", err)
	}
}

func TestFailedAddRouteForDNS64Network(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simAddRouteFail: true},
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		DNS64: lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
	}
	err := lazyjack.AddRouteForDNS64Network(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail adding IPv6 route")
	}
	expected := "mock fail add route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedExistingRouteForDNS64Network(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simRouteExists: true},
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		DNS64: lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
	}
	err := lazyjack.AddRouteForDNS64Network(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail adding IPv6 route")
	}
	expected := "skipping - add route to fd00:10:64:ff9b::/96 via fd00:10::200 as already exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestPrepareDNS64Server(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:    &MockHypervisor{simNotExists: true},
			WorkArea: workArea,
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
	}
	err := lazyjack.PrepareDNS64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to prepare DNS64 server: %v", err)
	}
}

func TestFailRunPrepareDNS64Server(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists: true,
				simRunFailed: true,
			},
			WorkArea: workArea,
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
	}
	err := lazyjack.PrepareDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail running DNS64 container")
	}
	expected := "mock fail to run container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedIPDeletePrepareDNS64Server(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists:           true,
				simDeleteInterfaceFail: true,
			},
			WorkArea: workArea,
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
	}
	err := lazyjack.PrepareDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail removing IPv4 address")
	}
	expected := "mock fail delete of IP"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRouteAddPrepareDNS64Server(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists:    true,
				simAddRouteFail: true,
			},
			WorkArea: workArea,
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
	}
	err := lazyjack.PrepareDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail adding IPv6 route")
	}
	expected := "mock fail add route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestNotExistsEnsureNAT64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNotExists: true},
		},
	}
	err := lazyjack.EnsureNAT64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to run NAT64 container: %v", err)
	}
}

func TestExistsButNotRunningEnsureNAT64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{},
		},
	}
	err := lazyjack.EnsureNAT64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to run NAT64 container: %v", err)
	}
}

func TestSkipRunningEnsureNAT64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simRunning: true},
		},
	}
	err := lazyjack.EnsureNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected NAT64 container to already be running")
	}
	expected := "skipping - NAT64 container (tayga) already running"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRemoveOldEnsureNAT64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteContainerFail: true,
			},
		},
	}
	err := lazyjack.EnsureNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail deleting old NAT64 container")
	}
	expected := "unable to remove existing (non-running) NAT64 container: mock fail delete of container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRunEnsureNAT64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simRunFailed: true,
			},
		},
	}
	err := lazyjack.EnsureNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail running NAT64 container")
	}
	expected := "mock fail to run container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestEnsureRouteToNAT64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
	}
	err := lazyjack.EnsureRouteToNAT64(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create routes: %v", err)
	}
}

func TestFailedEnsureRouteToNAT64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
	}
	err := lazyjack.EnsureRouteToNAT64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestSkippingEnsureRouteToNAT64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteExists: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
	}
	err := lazyjack.EnsureRouteToNAT64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected route to already exist")
	}
	expected := "skipping - add route to 172.18.0.128/25 via 172.18.0.200 as already exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestPrepareNAT64Server(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{simNotExists: true},
			NetMgr: nm,
		},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
	}
	err := lazyjack.PrepareNAT64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to prepare NAT64 container: %v", err)
	}
}

func TestFailRunPrepareNAT64Server(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteContainerFail: true,
			},
			NetMgr: nm,
		},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
	}
	err := lazyjack.PrepareNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail deleting old NAT64 container")
	}
	expected := "unable to remove existing (non-running) NAT64 container: mock fail delete of container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailRouteAddPrepareNAT64Server(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{},
			NetMgr: nm,
		},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
	}
	err := lazyjack.PrepareNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestPrepare(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	// Make needed files
	filename := filepath.Join(etcArea, lazyjack.EtcHostsFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create hosts file for test")
	}
	filename = filepath.Join(etcArea, lazyjack.EtcResolvConfFile)
	err = ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	systemdArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemdArea, t)
	defer HelperCleanupArea(systemdArea, t)

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				Name:          "master",
				ID:            10,
				Interface:     "eth1",
				IsNAT64Server: true,
				IsDNS64Server: true,
				IsMaster:      true,
			},
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists: true,
			},
			WorkArea:    workArea,
			EtcArea:     etcArea,
			SystemdArea: systemdArea,
			NetMgr:      nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{
			ServerIP:      "fd00:10::200",
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.32.0.0/16",
		},
	}
	err = lazyjack.Prepare("master", c)
	if err != nil {
		t.Fatalf("FAILED: Expected prepare to succeed: %v", err)
	}
}

func TestFailSupportNetCreatePrepare(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
			Hyper: &MockHypervisor{
				simNotExists:     true,
				simCreateNetFail: true,
			},
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
	}
	err := lazyjack.Prepare("master", c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail creating support network")
	}
	expected := "mock fail create of network"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailPrepDNS64Prepare(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
			Hyper: &MockHypervisor{
				simNotExists: true,
				simRunFailed: true,
			},
			WorkArea: workArea,
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
	}
	err := lazyjack.Prepare("master", c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail run of DNS64 service")
	}
	expected := "mock fail to run container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailPrepNAT64Prepare(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
			Hyper: &MockHypervisor{
				simNotExists: true,
			},
			WorkArea: workArea,
			NetMgr:   nm,
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{
			ServerIP:      "fd00:10::200",
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.32.0.0/16",
		},
	}
	err := lazyjack.Prepare("master", c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail prep of NAT64 route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailClusterNodePrepare(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	// Missing hosts file to cause failure

	systemdArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemdArea, t)
	defer HelperCleanupArea(systemdArea, t)

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				Name:          "master",
				ID:            10,
				Interface:     "eth1",
				IsNAT64Server: true,
				IsDNS64Server: true,
				IsMaster:      true,
			},
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simNotExists: true,
			},
			WorkArea:    workArea,
			EtcArea:     etcArea,
			SystemdArea: systemdArea,
			NetMgr:      nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{
			ServerIP:      "fd00:10::200",
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
		Support: lazyjack.SupportNetwork{
			Info: lazyjack.NetInfo{
				Prefix: "2001:db8:10::",
			},
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.32.0.0/16",
		},
	}
	err := lazyjack.Prepare("master", c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail prep node cluster")
	}
	expected := "unable to read"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}
