package lazyjack_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestKubeletDropInContents(t *testing.T) {
	c := &lazyjack.Config{
		Service: lazyjack.ServiceNetwork{CIDR: "2001:db8::/110"},
	}

	expected := `[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=2001:db8::a --cluster-domain=cluster.local"
`
	actual := lazyjack.CreateKubeletDropInContents(c)
	if actual.String() != expected {
		t.Errorf("Kubelet drop-in contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
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
		t.Errorf("FAILURE: Expected to be able to create drop-in file: %s", err.Error())
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
		t.Errorf("FAILURE: Expected not to be able to create area for drop-in file")
	}
}

func TestBuildFileStructureForDNS(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	err := lazyjack.BuildFileStructureForDNS(basePath)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create DNS area in %q: %s", basePath, err.Error())
	}
	conf := filepath.Join(basePath, lazyjack.DNS64BaseArea, lazyjack.DNS64ConfArea)
	if _, err := os.Stat(conf); os.IsNotExist(err) {
		t.Errorf("FAILED: Config area was not created")
	}
	cache := filepath.Join(basePath, lazyjack.DNS64BaseArea, lazyjack.DNS64CacheArea)
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		t.Errorf("FAILED: Cache area was not created")
	}
}

func TestFailingBuildFileStructureForDNS(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Make it not readable, so that it cannot be removed
	err := os.MkdirAll(basePath, 0400)
	if err != nil {
		t.Errorf("ERROR: Test setup failure: %s", err.Error())
	}

	err = lazyjack.BuildFileStructureForDNS(basePath)
	if err == nil {
		t.Errorf("FAILED: Expected not to be able to create DNS area in %q", basePath)
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
		t.Errorf("DNS64 named.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual.String())
	}
}

func TestCreateConfigForDNS64(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			RemoteV4Server: "8.8.8.8",
		},
		General: lazyjack.GeneralSettings{
			WorkArea: basePath,
		},
	}

	err := lazyjack.CreateConfigForDNS64(c)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create DNS64 config area and file: %s", err.Error())
	}
	conf := filepath.Join(c.General.WorkArea, lazyjack.DNS64BaseArea, lazyjack.DNS64ConfArea, lazyjack.DNS64NamedConf)
	if _, err := os.Stat(conf); os.IsNotExist(err) {
		t.Errorf("FAILED: Config file %q was not created", conf)
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
			t.Errorf("FAILED: [%s]. Expected %q, got %q", tc.name, tc.expected, actual)
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
			Prefix: "fd00:100::",
		},
	}

	ni := lazyjack.BuildNodeInfo(c)
	if len(ni) != 3 {
		t.Errorf("FAILURE: Expected three nodes")
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
			Prefix: "fd00:100::",
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
		t.Errorf("FAILED: Expected to be able to update hosts file: %s", err.Error())
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
			EtcArea: basePath,
		},
	}

	// Make a file to read
	filename := filepath.Join(basePath, lazyjack.EtcResolvConfFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	err = lazyjack.AddResolvConfEntry(c)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to update resolv.conf file: %s", err.Error())
	}
}

func TestFindHostIPForNAT64(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
			},
			"minion1": {
				ID:            20,
				IsNAT64Server: true,
			},
			"minion2": {
				ID:            30,
				IsNAT64Server: false,
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	gw, ok := lazyjack.FindHostIPForNAT64(c)
	if !ok {
		t.Errorf("Expected to find node with NAT64 server")
	}
	if gw != "fd00:100::20" {
		t.Errorf("Incorrect GW IP from node with NAT64 server")
	}
	bad := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
			},
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	gw, ok = lazyjack.FindHostIPForNAT64(bad)
	if ok {
		t.Errorf("Expected no NAT64 server to be found")
	}
}

func TestKubeAdmConfigContents(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token: "56cdce.7b18ad347f3de81c",
		},
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
	actual := string(lazyjack.CreateKubeAdmConfigContents(n, c))
	if actual != expected {
		t.Errorf("FAILED: kubeadm.conf contents wrong\nExpected: %s\n  Actual: %s\n", expected, actual)
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
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	err := lazyjack.CreateKubeAdmConfigFile(n, c)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create KubeAdm config file: %s", err.Error())
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
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name: "my-master",
		ID:   10,
	}

	HelperMakeReadOnly(basePath, t)

	err := lazyjack.CreateKubeAdmConfigFile(n, c)
	if err == nil {
		t.Errorf("FAILED: Expected not to be able to create KubeAdm config file")
	}
}
