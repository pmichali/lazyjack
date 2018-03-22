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
		Service: lazyjack.ServiceNetwork{CIDR: "2001:db8::/110"},
	}

	expected := `[Service]
Environment="KUBELET_DNS_ARGS=--cluster-dns=2001:db8::a --cluster-domain=cluster.local"
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

func TestBuildFileStructureForDNS(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	err := lazyjack.BuildFileStructureForDNS(basePath)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create DNS area in %q: %s", basePath, err.Error())
	}
	conf := filepath.Join(basePath, lazyjack.DNS64BaseArea, lazyjack.DNS64ConfArea)
	if _, err := os.Stat(conf); os.IsNotExist(err) {
		t.Fatalf("FAILED: Config area was not created")
	}
	cache := filepath.Join(basePath, lazyjack.DNS64BaseArea, lazyjack.DNS64CacheArea)
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		t.Fatalf("FAILED: Cache area was not created")
	}
}

func TestFailingBuildFileStructureForDNS(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Set work area a level lower, so that we can make parent read-only, preventing deletion
	workArea := filepath.Join(basePath, "dummy")
	err := os.MkdirAll(workArea, 0700)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	err = os.Chmod(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	defer func() { os.Chmod(basePath, 0700) }()

	err = lazyjack.BuildFileStructureForDNS(workArea)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create DNS area in %q", workArea)
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

func TestCreateSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNotExists: true},
		},
		Support: lazyjack.SupportNetwork{
			Prefix: "2001:db8:10::",
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
			Prefix: "2001:db8:10::",
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
			Prefix: "2001:db8:10::",
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
		t.Fatalf("FAILED: Expected to be able to create DNS64 config area and file: %s", err.Error())
	}
	conf := filepath.Join(c.General.WorkArea, lazyjack.DNS64BaseArea, lazyjack.DNS64ConfArea, lazyjack.DNS64NamedConf)
	if _, err := os.Stat(conf); os.IsNotExist(err) {
		t.Fatalf("FAILED: Config file %q was not created", conf)
	}
}

func TestFailedBuildTreeCreateConfigForDNS64(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Make it not readable, so that it cannot be removed
	err := os.MkdirAll(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}

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

	err = lazyjack.CreateConfigForDNS64(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create DNS64 config area")
	}
	expected := "unable to create directory structure for DNS64"
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
			Prefix: "fd00:100::",
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
		t.Fatalf("FAILED: Expected to be able to update resolv.conf file: %s", err.Error())
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
		t.Fatalf("Expected to find node with NAT64 server")
	}
	if gw != "fd00:100::20" {
		t.Fatalf("Incorrect GW IP from node with NAT64 server")
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
		t.Fatalf("Expected no NAT64 server to be found")
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
			Prefix: "fd00:100::",
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
		t.Fatalf("FAILED: Expected not to be able to create KubeAdm config file")
	}
}

func TestCreateRouteToNAT64ServerForDNS64SubnetForNATServer(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	c := &lazyjack.Config{
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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

func TestCreateRouteToNAT64ServerForDNS64SubnetForNonNATServer(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
			Prefix: "fd00:100::",
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
			Prefix: "fd00:100::",
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
	c := &lazyjack.Config{
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteExists: true}}
	c := &lazyjack.Config{
		DNS64:   lazyjack.DNS64Config{CIDR: "fd00:10:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "fd00:10::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
			Prefix: "fd00:100::",
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
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
			Prefix: "fd00:100::",
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteExists: true}}
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
			Prefix: "fd00:100::",
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
			Prefix: "fd00:100::",
			Size:   64,
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

	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
		},
		Support: lazyjack.SupportNetwork{
			CIDR:   "fd00:10::/64",
			V4CIDR: "172.20.0.0/16",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
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
	}
	err := lazyjack.EnsureDNS64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to run DNS64 container: %v", err)
	}
}

func TestExistsButNotRunningEnsureDNS64Server(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:    &MockHypervisor{},
			WorkArea: workArea,
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
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:    &MockHypervisor{simRunning: true},
			WorkArea: workArea,
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
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteContainerFail: true,
			},
			WorkArea: workArea,
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
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Set work area a level lower, so that we can make parent read-only, preventing deletion
	workArea := filepath.Join(basePath, "dummy")
	err := os.MkdirAll(workArea, 0700)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	err = os.Chmod(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	defer func() { os.Chmod(basePath, 0700) }()

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
	}
	err = lazyjack.EnsureDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail to create config for DNS64")
	}
	expected := "unable to create directory structure for DNS64"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRunEnsureDNS64Server(t *testing.T) {
	workArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(workArea, t)
	defer HelperCleanupArea(workArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simRunFailed: true,
			},
			WorkArea: workArea,
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteExists: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{NetMgr: nm},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{simNotExists: true},
			NetMgr: nm,
		},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteContainerFail: true,
			},
			NetMgr: nm,
		},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{},
			NetMgr: nm,
		},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
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

	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
			Prefix: "fd00:100::",
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
			Prefix: "2001:db8:10::",
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
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
			Hyper: &MockHypervisor{
				simNotExists:     true,
				simCreateNetFail: true,
			},
		},
		Support: lazyjack.SupportNetwork{
			Prefix: "2001:db8:10::",
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
			Prefix: "2001:db8:10::",
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

	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
				IsDNS64Server: true,
			},
		},
		General: lazyjack.GeneralSettings{
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
			Prefix: "2001:db8:10::",
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
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

	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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
			Prefix: "fd00:100::",
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
			Prefix: "2001:db8:10::",
			CIDR:   "2001:db8:10::/64",
			V4CIDR: "172.20.0.0/16",
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
