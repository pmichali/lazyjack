package lazyjack_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestRestoreEtcHostsContents(t *testing.T) {
	var testCases = []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "restore old, delete new",
			input: bytes.NewBufferString(`# restore old, remove new
#[-] 10.0.0.2 master
#[-] 10.0.0.3 minion
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`).Bytes(),
			expected: `# restore old, remove new
10.0.0.2 master
10.0.0.3 minion
`,
		},
		{
			name: "Ignore commented, remove new",
			input: bytes.NewBufferString(`# ignore commented
#[-] 10.0.0.2 master
# 10.0.0.3 minion
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`).Bytes(),
			expected: `# ignore commented
10.0.0.2 master
# 10.0.0.3 minion
`,
		},
		{
			name: "Remove new, no existing",
			input: bytes.NewBufferString(`# remove new
127.0.0.1 localhost
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`).Bytes(),
			expected: `# remove new
127.0.0.1 localhost
`,
		},
		{
			name: "No new",
			input: bytes.NewBufferString(`# no new
10.0.0.2 master
fd00:20::10 master
fd00:20::20 minion
`).Bytes(),
			expected: `# no new
10.0.0.2 master
fd00:20::10 master
fd00:20::20 minion
`,
		},
		{
			name: "Retore multiple old",
			input: bytes.NewBufferString(`# restore multiple old
#[-] 10.0.0.2 master
#[-] 10.0.0.3 minion
#[-] 10.0.0.2 master
#[-] 10.0.0.3 minion
fd00:20::10 master  #[+]
fd00:20::20 minion  #[+]
`).Bytes(),
			expected: `# restore multiple old
10.0.0.2 master
10.0.0.3 minion
10.0.0.2 master
10.0.0.3 minion
`,
		},
	}
	for _, tc := range testCases {
		actual := lazyjack.RevertConfigInfo(tc.input, "test-file")
		if string(actual) != tc.expected {
			t.Errorf("FAILED: [%s] mismatch. Expected:\n%s\nActual:\n%s\n", tc.name, tc.expected, string(actual))
		}
	}
}

func TestRestoreResolvConfContents(t *testing.T) {
	var testCases = []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "no nameservers",
			input: bytes.NewBufferString(`# no nameservers
search example.com
nameserver fd00:10::100  #[+]
`).Bytes(),
			expected: `# no nameservers
search example.com
`,
		},
		{
			name: "remove new",
			input: bytes.NewBufferString(`# remove new
search example.com
nameserver fd00:10::100  #[+]
nameserver 8.8.8.8
nameserver 8.8.4.4
`).Bytes(),
			expected: `# remove new
search example.com
nameserver 8.8.8.8
nameserver 8.8.4.4
`,
		},
		{
			name: "revert position",
			input: bytes.NewBufferString(`# revert position
search example.com
nameserver fd00:10::100  #[+]
nameserver 8.8.8.8
#[-] nameserver fd00:10::100
nameserver 8.8.4.4
`).Bytes(),
			expected: `# revert position
search example.com
nameserver 8.8.8.8
nameserver fd00:10::100
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
	}

	for _, tc := range testCases {
		actual := lazyjack.RevertConfigInfo(tc.input, "test-file")
		if string(actual) != tc.expected {
			t.Errorf("FAILED: [%s] mismatch.\nExpected:\n%s\nActual:\n%s\n", tc.name, tc.expected, string(actual))
		}
	}

}

func TestRevertEntries(t *testing.T) {
	srcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(srcArea, t)
	defer HelperCleanupArea(srcArea, t)

	// Create a valid source file
	src := filepath.Join(srcArea, "foo")
	err := ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}
	backup := filepath.Join(srcArea, "foo.back")

	err = lazyjack.RevertEntries(src, backup)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to restore entry: %s", err.Error())
	}
}

func TestFailingRevertEntries(t *testing.T) {
	srcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(srcArea, t)
	defer HelperCleanupArea(srcArea, t)

	src := filepath.Join(srcArea, "foo")
	backup := filepath.Join(srcArea, "foo.bak")

	err := lazyjack.RevertEntries(src, backup)
	if err == nil {
		t.Fatalf("FAILED: Expected to NOT be able to restore entry - missing source")
	}
	expected := "unable to read"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected reason to start with %q, got %q", expected, err.Error())
	}

	// Create a valid source file
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	// Use directory as backup file, so rename fails
	backup = srcArea
	err = lazyjack.RevertEntries(src, backup)
	if err == nil {
		t.Fatalf("FAILED: Expected to NOT be able to restore entry - read-only backup")
	}
	expected = "unable to backup"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected reason to start with %q, got %q", expected, err.Error())
	}
}

func TestRemoveDropInFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	// Create a dummy file to delete
	src := filepath.Join(basePath, lazyjack.KubeletDropInFile)
	err := ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{SystemdArea: basePath},
	}
	err = lazyjack.RemoveDropInFile(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove kubelet drop in file: %s", err.Error())
	}
}

func TestFailedNoFileRemoveDropInFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{SystemdArea: basePath},
	}
	err := lazyjack.RemoveDropInFile(c)
	if err == nil {
		t.Fatalf("FAILED: Expected kubelet drop in file to be missing")
	}
	expected := "no kubelet drop-in file to remove"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestFailedRemoveDropInFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Set fake systemd area a level lower, so that we can make parent read-only, preventing deletion
	systemdBase := filepath.Join(basePath, "dummy")
	err := os.MkdirAll(systemdBase, 0700)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	// Create a dummy file that is not writeable
	src := filepath.Join(systemdBase, lazyjack.KubeletDropInFile)
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0400)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	// Make parent dir read-only, to prevent file removal for test.
	err = os.Chmod(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	defer func() { os.Chmod(basePath, 0700) }()

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{SystemdArea: systemdBase},
	}
	err = lazyjack.RemoveDropInFile(c)
	if err == nil {
		t.Fatalf("FAILED: Expected kubelet drop in file removal to fail")
	}
	expected := "unable to remove kubelet drop-in file"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected reason to start with %q, got %q", expected, err.Error())
	}
}

func TestRemoveManagementIP(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
					Size:   64,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Interface: "eth1",
		ID:        0x10,
	}

	err := lazyjack.RemoveManagementIP(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove management IP: %s", err.Error())
	}
}

func TestFailedRemoveManagementIP(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simDeleteFail: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
					Size:   64,
				},
			},
		},
	}
	n := &lazyjack.Node{
		Interface: "eth1",
		ID:        0x10,
	}

	err := lazyjack.RemoveManagementIP(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to remove management IP")
	}
	expected := "unable to remove IP from management interface: unable to delete ip \"2001:db8:20::10/64\" from interface \"eth1\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestRevertFile(t *testing.T) {
	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{EtcArea: etcArea},
	}

	// Create a valid source file in area
	src := filepath.Join(etcArea, "foo")
	err := ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	err = lazyjack.RevertEtcAreaFile(c, "foo", "foo.bak")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to revert file: %s", err.Error())
	}
}

func TestRemoveRouteForDNS64ForNAT64Node(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
				},
			},
		},
		DNS64:   lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "2001:db8:5::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
	}
	n := &lazyjack.Node{
		Interface:     "eth1",
		IsNAT64Server: true,
		ID:            0x10,
	}
	err := lazyjack.RemoveRouteForDNS64(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove route: %s", err.Error())
	}
}

func TestFailedRemoveRouteForDNS64ForNAT64Node(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteDelFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
				},
			},
		},
		DNS64:   lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "2001:db8:5::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
	}
	n := &lazyjack.Node{
		Interface:     "eth1",
		IsNAT64Server: true,
		ID:            0x10,
	}
	err := lazyjack.RemoveRouteForDNS64(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to remove route")
	}
	expected := "unable to delete route to 2001:db8:64:ff9b::/96 via 2001:db8:5::200: mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestRemoveRouteForDNS64ForNonNAT64Node(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:40::",
				},
			},
		},
		DNS64:   lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "2001:db8:6::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.32.0.0/16"},
	}
	n := &lazyjack.Node{
		Interface:     "eth1",
		IsNAT64Server: false,
		ID:            10,
	}
	err := lazyjack.RemoveRouteForDNS64(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove route: %s", err.Error())
	}
}

func TestFailedNoNatRemoveRouteForDNS64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
			},
			"minion": {
				ID:            20,
				IsNAT64Server: false,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:40::",
				},
			},
		},
		DNS64:   lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64:   lazyjack.NAT64Config{ServerIP: "2001:db8:6::200"},
		Support: lazyjack.SupportNetwork{V4CIDR: "172.20.0.0/16"},
	}
	n := &lazyjack.Node{
		Interface:     "eth1",
		IsNAT64Server: false,
		ID:            10,
	}
	err := lazyjack.RemoveRouteForDNS64(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to find NAT64 server")
	}
	expected := "unable to delete route to 2001:db8:64:ff9b::/96 via : unable to find node with NAT64 server"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestRemoveRouteForNAT64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
				},
			},
		},
		Support: lazyjack.SupportNetwork{CIDR: "2001:db8::/64"},
	}
	n := &lazyjack.Node{
		Interface: "eth1",
		ID:        20,
	}
	err := lazyjack.RemoveRouteForNAT64(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove route: %s", err.Error())
	}
}

func TestFailedRemoveRouteForNAT64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteDelFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
				},
			},
		},
		Support: lazyjack.SupportNetwork{CIDR: "2001:db8::/64"},
	}
	n := &lazyjack.Node{
		Interface: "eth1",
		ID:        0x20,
	}
	err := lazyjack.RemoveRouteForNAT64(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to remove route")
	}
	expected := "unable to delete route to 2001:db8::/64 via 2001:db8:20::10: mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestFailedNoNATRemoveRouteForNAT64(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            10,
				IsNAT64Server: false,
			},
			"minion": {
				ID:            20,
				IsNAT64Server: false,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
				},
			},
		},
		Support: lazyjack.SupportNetwork{CIDR: "2001:db8::/64"},
	}
	n := &lazyjack.Node{
		Interface: "eth1",
		ID:        20,
	}
	err := lazyjack.RemoveRouteForNAT64(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to remove route")
	}
	expected := "unable to delete route to 2001:db8::/64 via : unable to find node with NAT64 server configured"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestCleanupClusterNode(t *testing.T) {
	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	// Create hosts and resolv.conf files
	src := filepath.Join(etcArea, lazyjack.EtcHostsFile)
	err := ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create hosts file for test")
	}
	src = filepath.Join(etcArea, lazyjack.EtcResolvConfFile)
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	systemArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemArea, t)
	defer HelperCleanupArea(systemArea, t)

	src = filepath.Join(systemArea, lazyjack.KubeletDropInFile)
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create drop-in file for test")
	}

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr:      nm,
			EtcArea:     etcArea,
			SystemdArea: systemArea,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
					Size:   64,
				},
			},
		},
		DNS64: lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64: lazyjack.NAT64Config{ServerIP: "2001:db8:5::200"},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.20.0.0/16",
			CIDR:   "2001:db8::/64",
		},
	}
	// Make sure this node is not the NAT64 or DNS64 server node
	n := &lazyjack.Node{
		Interface:     "eth2",
		IsNAT64Server: false,
		ID:            0x20,
	}

	err = lazyjack.CleanupClusterNode(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to clean up cluster: %s", err.Error())
	}
}

func TestFailedCleanupClusterNode(t *testing.T) {
	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	// Missing hosts and resolv.conf files

	systemArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemArea, t)
	defer HelperCleanupArea(systemArea, t)

	// Missing drop-in file

	// Simulating address delete and all route deletes fail
	nm := lazyjack.NetMgr{Server: mockNetLink{
		simRouteDelFail: true,
		simDeleteFail:   true,
	}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr:      nm,
			EtcArea:     etcArea,
			SystemdArea: systemArea,
			Mode:        lazyjack.IPv6NetMode,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
					Size:   64,
				},
			},
		},
		DNS64: lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64: lazyjack.NAT64Config{ServerIP: "2001:db8:5::200"},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.32.0.0/16",
			CIDR:   "2001:db8::/64",
		},
	}
	// Make sure this node is not the NAT64 or DNS64 server node
	n := &lazyjack.Node{
		Interface:     "eth2",
		IsNAT64Server: false,
		ID:            0x20,
	}

	err := lazyjack.CleanupClusterNode(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to have multiple failures cleaning up node")
	}
	// Check that each error message is seen
	actual := strings.Split(err.Error(), ". ")
	if len(actual) != 6 {
		t.Fatalf("FAILED: Expected 6 error strings, got %d", len(actual))
	}
	expected := []*regexp.Regexp{
		regexp.MustCompile("no kubelet drop-in file to remove"),
		regexp.MustCompile("unable to remove IP from management interface: unable to delete ip \"2001:db8:20::20/64\" from interface \"eth2\""),
		regexp.MustCompile("unable to read file .+/hosts to revert: unable to read .+/hosts: open .+/hosts: no such file or directory"),
		regexp.MustCompile("unable to read file .+/resolv.conf to revert: unable to read .+/resolv.conf: open .+/resolv.conf: no such file or directory"),
		regexp.MustCompile("unable to delete route to 2001:db8:64:ff9b::/96 via 2001:db8:20::10: mock failure deleting route"),
		regexp.MustCompile("unable to delete route to 2001:db8::/64 via 2001:db8:20::10: mock failure deleting route"),
	}
	for i, _ := range actual {
		if !expected[i].MatchString(actual[i]) {
			t.Errorf("FAILED: Expected reason to match pattern %q, got %q", expected[i].String(), actual[i])
		}
	}
}

func TestRemoveContainer(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{},
		},
	}
	err := lazyjack.RemoveContainer("foo", c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove container: %v", err)
	}
}

func TestFailedRemoveContainer(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simDeleteContainerFail: true},
		},
	}
	err := lazyjack.RemoveContainer("foo", c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to remove container")
	}
	expected := "unable to remove \"foo\" container: mock fail delete of container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestSkippedRemoveContainer(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNotExists: true},
		},
	}
	err := lazyjack.RemoveContainer("foo", c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to remove non-existing container")
	}
	expected := "skipping - No \"foo\" container exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestCleanupDNS64Server(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:    &MockHypervisor{},
			WorkArea: basePath,
		},
	}
	err := lazyjack.CleanupDNS64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to clean up DNS64 server: %v", err)
	}
}

func TestFailedDeleteCleanupDNS64Server(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:    &MockHypervisor{simDeleteContainerFail: true},
			WorkArea: basePath,
		},
	}
	err := lazyjack.CleanupDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete DNS64 server")
	}
	expected := "unable to remove \"bind9\" container: mock fail delete of container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestFailedDeleteVolumeCleanupDNS64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simDeleteVolumeFail: true},
		},
	}
	err := lazyjack.CleanupDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete volume")
	}
	expected := "mock fail delete of volume"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestSkippedDeleteVolumeCleanupDNS64Server(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNotExists: true}, // Because if this setting, no container will exist either
		},
	}
	err := lazyjack.CleanupDNS64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to skip volume delete")
	}
	expected := "skipping - No \"bind9\" container exists. skipping - No \"volume-bind9\" volume"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestCleanupNAT64Server(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{},
			NetMgr: nm,
		},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.32.0.0/16",
		},
	}
	err := lazyjack.CleanupNAT64Server(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to clean up NAT64 server: %v", err)
	}
}

func TestFailedDeleteCleanupNAT64Server(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{simDeleteContainerFail: true},
			NetMgr: nm,
		},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.32.0.0/16",
		},
	}
	err := lazyjack.CleanupNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete NAT64 server")
	}
	expected := "unable to remove \"tayga\" container: mock fail delete of container"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestFailedDeleteRouteCleanupNAT64Server(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteDelFail: true}}
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper:  &MockHypervisor{},
			NetMgr: nm,
		},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
		},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.32.0.0/16",
		},
	}
	err := lazyjack.CleanupNAT64Server(c)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete route for NAT64 server")
	}
	expected := "mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestCleanupSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{},
		},
	}

	err := lazyjack.CleanupSupportNetwork(c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to cleanup support network: %s", err.Error())
	}
}

func TestSkippedNonExistsCleanupSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simNotExists: true},
		},
	}
	err := lazyjack.CleanupSupportNetwork(c)
	if err == nil {
		t.Fatalf("FAILED: Expected support network to not exist")
	}
	expected := "skipping - support network does not exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestFailedCleanupSupportNetwork(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{simDeleteNetFail: true},
		},
	}

	err := lazyjack.CleanupSupportNetwork(c)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail deleting support network")
	}
	expected := "unable to remove support network: mock fail delete of network"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}

func TestCleanup(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	// Create hosts and resolv.conf files
	src := filepath.Join(etcArea, lazyjack.EtcHostsFile)
	err := ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create hosts file for test")
	}
	src = filepath.Join(etcArea, lazyjack.EtcResolvConfFile)
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	systemArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemArea, t)
	defer HelperCleanupArea(systemArea, t)

	src = filepath.Join(systemArea, lazyjack.KubeletDropInFile)
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create drop-in file for test")
	}

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
				IsDNS64Server: true,
				IsMaster:      true,
				Interface:     "eth1",
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			Hyper:       &MockHypervisor{},
			NetMgr:      nm,
			WorkArea:    basePath,
			EtcArea:     etcArea,
			SystemdArea: systemArea,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
					Size:   64,
				},
			},
		},
		DNS64: lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
			ServerIP:      "2001:db8:5::200",
		},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.32.0.0/16",
			CIDR:   "2001:db8::/64",
		},
	}

	err = lazyjack.Cleanup("master", c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to clean up: %v", err)
	}
}

func TestFailedCleanup(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	etcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(etcArea, t)
	defer HelperCleanupArea(etcArea, t)

	// Create hosts and resolv.conf files
	src := filepath.Join(etcArea, lazyjack.EtcHostsFile)
	err := ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create hosts file for test")
	}
	src = filepath.Join(etcArea, lazyjack.EtcResolvConfFile)
	err = ioutil.WriteFile(src, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create resolv.conf file for test")
	}

	systemArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(systemArea, t)
	defer HelperCleanupArea(systemArea, t)

	// Missing drop-in file in system area to cause failure

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID:            0x10,
				IsNAT64Server: true,
				IsDNS64Server: true,
				IsMaster:      true,
				Interface:     "eth1",
			},
			"minion": {
				ID: 0x20,
			},
		},
		General: lazyjack.GeneralSettings{
			Hyper: &MockHypervisor{
				simDeleteContainerFail: true,
				simDeleteNetFail:       true,
			},
			NetMgr:      nm,
			WorkArea:    basePath,
			EtcArea:     etcArea,
			SystemdArea: systemArea,
			Mode:        lazyjack.IPv6NetMode,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "2001:db8:20::",
					Size:   64,
				},
			},
		},
		DNS64: lazyjack.DNS64Config{CIDR: "2001:db8:64:ff9b::/96"},
		NAT64: lazyjack.NAT64Config{
			V4MappingCIDR: "172.18.0.128/25",
			V4MappingIP:   "172.18.0.200",
			ServerIP:      "2001:db8:5::200",
		},
		Support: lazyjack.SupportNetwork{
			V4CIDR: "172.32.0.0/16",
			CIDR:   "2001:db8::/64",
		},
	}

	err = lazyjack.Cleanup("master", c)
	if err == nil {
		t.Fatalf("FAILED: Expected to have failure cleaning up")
	}

	actual := strings.Split(err.Error(), ". ")
	if len(actual) != 4 {
		t.Fatalf("FAILED: Expected 4 error strings, got %d (%v)", len(actual), actual)
	}
	expected := []*regexp.Regexp{
		regexp.MustCompile("no kubelet drop-in file to remove"),
		regexp.MustCompile("unable to remove \"bind9\" container: mock fail delete of container"),
		regexp.MustCompile("unable to remove \"tayga\" container: mock fail delete of container"),
		regexp.MustCompile("unable to remove support network: mock fail delete of network"),
	}
	for i, _ := range actual {
		if !expected[i].MatchString(actual[i]) {
			t.Errorf("FAILED: Expected reason to match pattern %q, got %q", expected[i].String(), actual[i])
		}
	}
}
