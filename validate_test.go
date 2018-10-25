package lazyjack_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestValidateCommand(t *testing.T) {
	var testCases = []struct {
		name        string
		command     string
		expected    string
		expectedStr string
	}{
		{
			name:        "Missing command",
			command:     "",
			expected:    "",
			expectedStr: "missing command",
		},
		{
			name:        "Unknown command",
			command:     "foo",
			expected:    "",
			expectedStr: "unknown command \"foo\"",
		},
		{
			name:        "Valid command",
			command:     "up",
			expected:    "up",
			expectedStr: "",
		},
		{
			name:        "Mixed case command",
			command:     "PrEPaRe",
			expected:    "prepare",
			expectedStr: "",
		},
	}

	for _, tc := range testCases {
		command, err := lazyjack.ValidateCommand(tc.command)
		if command != tc.expected {
			t.Errorf("[%s] Expected %s, got %s", tc.name, tc.expected, command)
		}
		if tc.expectedStr != "" {
			actualErrStr := err.Error()
			if actualErrStr != tc.expectedStr {
				t.Errorf("[%s] Expected error string %q, got %q", tc.name, tc.expectedStr, actualErrStr)
			}
		}
	}
}

func TestValidateConfigFile(t *testing.T) {
	// No file specified
	cf, err := lazyjack.OpenConfigFile("")
	if cf != nil {
		t.Fatalf("Did not expect to have config, with empty filename")
	}
	if err.Error() != "unable to open config file \"\": open : no such file or directory" {
		t.Fatalf("Expected error message, when trying to open empty filename")
	}

	// File does not exist
	cf, err = lazyjack.OpenConfigFile("non-existing-file")
	if cf != nil {
		t.Fatalf("Did not expect to have config, with non-existing filename")
	}
	if err.Error() != "unable to open config file \"non-existing-file\": open non-existing-file: no such file or directory" {
		t.Fatalf("Expected error message, when trying to open non-existing filename")
	}

	// Have a file
	cf, err = lazyjack.OpenConfigFile("sample-config.yaml")
	if cf == nil {
		t.Fatalf("Expected to open sample configuration file")
	} else {
		cf.Close()
	}
}

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	// NOP to satisfy the Close method for io.ReadCloser
	// A bytes.Buffer already implements Read method
	return nil
}

func TestFailedBadYAMLLoadConfig(t *testing.T) {
	badYAML := `# Simple YAML file with (invalid) tab character
topology:
    good-host:
       interface: "eth0"
\topmodes: "master"
       id: 2`
	stream := &ClosingBuffer{bytes.NewBufferString(badYAML)}
	config, err := lazyjack.LoadConfig(stream)
	if config != nil {
		t.Fatalf("Should not have config loaded, if YAML is malformed")
	}
	if err.Error() != "Failed to parse config: yaml: line 5: did not find expected key" {
		t.Fatalf("Error message is not correct for malformed YAML file (%s)", err.Error())
	}
}

func TestLegacyLoadConfig(t *testing.T) {
	legacyYAML := `# Valid legacy yaml
plugin: bridge
`
	stream := &ClosingBuffer{bytes.NewBufferString(legacyYAML)}
	config, err := lazyjack.LoadConfig(stream)

	if err != nil {
		t.Fatalf("Unexpected error, when reading config")
	}
	if config == nil {
		t.Fatalf("Should have a config")
	}
	// Checking legacy location
	if config.Plugin != "bridge" {
		t.Fatalf("Missing plugin config")
	}
}

func TestLoadConfig(t *testing.T) {
	goodYAML := `# Valid YAML file
general:
    plugin: bridge
    work-area: "/override/path/for/work/area"
topology:
    my-master:
        interface: "eth0"
        opmodes: "master dns64 nat64"
        id: 2
    my-minion:
        interface: "enp10s0"
        opmodes: "minion"
        id: 3
support_net:
    cidr: "fd00:10::/64"
mgmt_net:
    cidr: "fd00:20::/64"
nat64:
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
    ip: "fd00:10::200"
dns64:
    remote_server: "8.8.8.8"  # Could be a internal/company DNS server
    cidr: "fd00:10:64:ff9b::/96"
    ip: "fd00:10::100"
    allow_aaaa_use: true`

	stream := &ClosingBuffer{bytes.NewBufferString(goodYAML)}
	config, err := lazyjack.LoadConfig(stream)

	if err != nil {
		t.Fatalf("Unexpected error, when reading config")
	}
	if config == nil {
		t.Fatalf("Should have a config")
	}

	if config.General.Plugin != "bridge" {
		t.Fatalf("Missing plugin config")
	}
	expected := "/override/path/for/work/area"
	if config.General.WorkArea != expected {
		t.Fatalf("Override to work area is incorrect. Expected %q, got %q", expected, config.General.WorkArea)
	}
	node, ok := config.Topology["my-master"]
	if !ok {
		t.Fatalf("Expected to have configuration for my-master node")
	}
	if node.Interface != "eth0" || node.ID != 2 || node.OperatingModes != "master dns64 nat64" {
		t.Fatalf("Incorrect config for node my-master (%+v)", node)
	}

	node, ok = config.Topology["my-minion"]
	if !ok {
		t.Errorf("Expected to have configuration for my-minion node")
	}
	if node.Interface != "enp10s0" || node.ID != 3 || node.OperatingModes != "minion" {
		t.Errorf("Incorrect config for node my-minion (%+v)", node)
	}

	if config.Support.CIDR != "fd00:10::/64" {
		t.Errorf("Support net config parse failed (%+v)", config.Support)
	}

	if config.Mgmt.CIDR != "fd00:20::/64" {
		t.Errorf("Management net config parse failed (%+v)", config.Mgmt)
	}

	if config.NAT64.V4MappingCIDR != "172.18.0.128/25" ||
		config.NAT64.V4MappingIP != "172.18.0.200" ||
		config.NAT64.ServerIP != "fd00:10::200" {
		t.Errorf("NAT64 config parse failure (%+v)", config.NAT64)
	}

	if config.DNS64.RemoteV4Server != "8.8.8.8" ||
		config.DNS64.CIDR != "fd00:10:64:ff9b::/96" ||
		config.DNS64.ServerIP != "fd00:10::100" ||
		!config.DNS64.AllowAAAAUse {
		t.Errorf("DNS64 config parse failure (%+v)", config.DNS64)
	}
}

func TestNotUniqueIDs(t *testing.T) {
	// Create bare minimum to test IDs
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID: 10,
			},
			"minion1": {
				ID: 20,
			},
			"minion2": {
				ID: 10,
			},
		},
	}
	err := lazyjack.ValidateUniqueIDs(c)
	if err == nil {
		t.Fatalf("Expected failure with duplicate IDs")
	}
	// Order of node names is not guaranteed, so just check first part of msg
	if !strings.HasPrefix(err.Error(), "duplicate node ID 10 seen for node") {
		t.Fatalf("Error message is not correct (%s)", err.Error())
	}
}

func TestUniqueIDs(t *testing.T) {
	// Create bare minimum to test IDs
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID: 10,
			},
			"minion1": {
				ID: 20,
			},
		},
	}
	err := lazyjack.ValidateUniqueIDs(c)
	if err != nil {
		t.Fatalf("Should not be an error, as all IDs are unique")
	}
}

func TestOperatingModesOnNode(t *testing.T) {
	var testCases = []struct {
		name        string
		opMode      string
		expectedStr string
	}{
		{
			name:        "Master only",
			opMode:      "master",
			expectedStr: "",
		},
		{
			name:        "Minion only",
			opMode:      "minion",
			expectedStr: "",
		},
		{
			name:        "Duplicates OK",
			opMode:      "nat64 minion dns64 minion",
			expectedStr: "",
		},
		{
			name:        "Master with DNS and NAT",
			opMode:      "master Dns64 NAT64",
			expectedStr: "",
		},
		{
			name:        "DNS/NAT and Master",
			opMode:      "dns64 nat64 Master",
			expectedStr: "",
		},
		{
			name:        "DNS and NAT only",
			opMode:      "nat64 dns64",
			expectedStr: "",
		},
		{
			name:        "No modes specified",
			opMode:      "",
			expectedStr: "missing operating mode for \"test-node\"",
		},
		{
			name:        "No modes specified",
			opMode:      "   ",
			expectedStr: "missing operating mode for \"test-node\"",
		},
		{
			name:        "Unknown mode",
			opMode:      "monster",
			expectedStr: "invalid operating mode \"monster\" for \"test-node\"",
		},
		{
			name:        "Master and minion",
			opMode:      "minion master",
			expectedStr: "invalid combination of modes for \"test-node\"",
		},
		// Don't currently support just DNS or just NAT
		// TODO: Decide if should allow DNS only/NAT only
		{
			name:        "Missing DNS64",
			opMode:      "nat64",
			expectedStr: "missing \"dns64\" mode for \"test-node\"",
		},
		{
			name:        "Missing NAT64",
			opMode:      "dns64",
			expectedStr: "missing \"nat64\" mode for \"test-node\"",
		},
	}

	for _, tc := range testCases {
		node := &lazyjack.Node{Name: "test-node", OperatingModes: tc.opMode}
		err := lazyjack.ValidateNodeOpModes(lazyjack.IPv6NetMode, node)
		if tc.expectedStr == "" {
			if err != nil {
				t.Errorf("[%s] Expected test to pass - see error: %s", tc.name, err.Error())
			}
		} else {
			if err == nil {
				t.Errorf("[%s] Expected test to fail", tc.name)
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}

func TestDuplicateMasters(t *testing.T) {
	// Create minimum to test node entries
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"node1": {
				OperatingModes: "master dns64 nat64",
			},
			"node2": {
				OperatingModes: "minion",
			},
			"node3": {
				OperatingModes: "master",
			},
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.ValidateOpModesForAllNodes(c)
	if err == nil {
		t.Fatalf("Expected to see error, when configuration has duplicate master nodes")
	} else if err.Error() != "found multiple nodes with \"master\" operating mode" {
		t.Fatalf("Duplicate master nodes error message wrong (%s)", err.Error())
	}
}

func TestNoMasterNode(t *testing.T) {
	// Create minimum to test node entries
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"node1": {
				OperatingModes: "dns64 nat64",
			},
			"node2": {
				OperatingModes: "minion",
			},
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.ValidateOpModesForAllNodes(c)
	if err == nil {
		t.Fatalf("Expected to see error, when configuration has no master node entry")
	} else if err.Error() != "no master node configuration" {
		t.Fatalf("No master node error message wrong (%s)", err.Error())
	}
}

func TestUnableToValidateOpModes(t *testing.T) {
	// Create minimum to test node entries
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"node1": {
				OperatingModes: "monster dns64 nat64",
			},
			"node2": {
				OperatingModes: "minion",
			},
		},
	}

	err := lazyjack.ValidateOpModesForAllNodes(c)
	expectedErr := "invalid operating mode \"monster\" for \"node1\""
	if err == nil {
		t.Fatalf("Expected to see error, when configuration has invalid op mode")
	} else if err.Error() != expectedErr {
		t.Fatalf("Have error %q, expected %q", err.Error(), expectedErr)
	}
}

func TestBootstrapToken(t *testing.T) {
	var testCases = []struct {
		name      string
		input     string
		errString string
	}{
		{
			name:      "Valid token",
			input:     "7aee33.05f81856d78346bd",
			errString: "",
		},
		{
			name:      "Missing/empty token",
			input:     "",
			errString: "missing token in config file",
		},
		{
			name:      "Wrong length",
			input:     "7aee33.05f81856d78346b",
			errString: "invalid token length (22)",
		},
		{
			name:      "Invalid value",
			input:     "ABCDEF.hasbadcharacters",
			errString: "token is invalid \"ABCDEF.hasbadcharacters\"",
		},
	}
	ignoreMissing := false
	for _, tc := range testCases {
		err := lazyjack.ValidateToken(tc.input, ignoreMissing)
		if err == nil {
			if tc.errString != "" {
				t.Errorf("[%s]: Expected error getting token: %s", tc.name, tc.errString)
			}
		} else {
			if err.Error() != tc.errString {
				t.Errorf("[%s]: Have error %q, expected %q", tc.name, err.Error(), tc.errString)
			}
		}
	}
}

func TestTokenCertificateHash(t *testing.T) {
	var testCases = []struct {
		name      string
		input     string
		errString string
	}{
		{
			name:      "Valid cert hash",
			input:     "123456789012345678901234567890123456789012345678901234567890abcd",
			errString: "",
		},
		{
			name:      "Missing/empty cert hash",
			input:     "",
			errString: "missing token certificate hash in config file",
		},
		{
			name:      "Wrong length",
			input:     "123456789012345678901234567890123456789012345678901234567890abc",
			errString: "invalid token certificate hash length (63)",
		},
		{
			name:      "Invalid value",
			input:     "123456789012345678901234567890123456789012345678hasbadcharacters",
			errString: "token certificate hash is invalid \"123456789012345678901234567890123456789012345678hasbadcharacters\"",
		},
	}
	ignoreMissing := false
	for _, tc := range testCases {
		err := lazyjack.ValidateTokenCertHash(tc.input, ignoreMissing)
		if err == nil {
			if tc.errString != "" {
				t.Errorf("[%s]: Expected error getting cert hash: %s", tc.name, tc.errString)
			}
		} else {
			if err.Error() != tc.errString {
				t.Errorf("[%s]: Have error %q, expected %q", tc.name, err.Error(), tc.errString)
			}
		}
	}
}

func TestGetNetAndMask(t *testing.T) {
	var testCases = []struct {
		name           string
		input          string
		expectedPrefix string
		expectedMask   int
		errString      string
	}{
		{name: "Valid CIDR",
			input:          "fd00:20::/64",
			expectedPrefix: "fd00:20::",
			expectedMask:   64,
			errString:      "",
		},
		{name: "Valid DNS CIDR",
			input:          "fd00:10:64:ff9b::/96",
			expectedPrefix: "fd00:10:64:ff9b::",
			expectedMask:   96,
			errString:      "",
		},
		{name: "Missing subnet prefix",
			input:          "/64",
			expectedPrefix: "",
			expectedMask:   0,
			errString:      "invalid CIDR address: /64",
		},
		{name: "Missing mask part",
			input:          "fd00:20::",
			expectedPrefix: "",
			expectedMask:   0,
			errString:      "invalid CIDR address: fd00:20::",
		},
		{name: "Missing mask part value",
			input:          "fd00:20::/",
			expectedPrefix: "",
			expectedMask:   0,
			errString:      "invalid CIDR address: fd00:20::/",
		},
		{name: "Invalid mask",
			input:          "fd00:20::/200",
			expectedPrefix: "",
			expectedMask:   0,
			errString:      "invalid CIDR address: fd00:20::/200",
		},
		{name: "Invalid subnet prefix",
			input:          "fd00::20::/64",
			expectedPrefix: "",
			expectedMask:   0,
			errString:      "invalid CIDR address: fd00::20::/64",
		},
	}
	for _, tc := range testCases {
		actualPrefix, actualMask, err := lazyjack.GetNetAndMask(tc.input)
		if err == nil {
			if tc.errString != "" {
				t.Errorf("[%s] Expected error (%s), but was successful converting", tc.name, tc.errString)
			} else {
				if actualPrefix != tc.expectedPrefix || actualMask != tc.expectedMask {
					t.Errorf("[%s} Conversion failed. Expected {%s %d}, got {%s %d}",
						tc.name, tc.expectedPrefix, tc.expectedMask, actualPrefix, actualMask)

				}
			}
		} else if err.Error() != tc.errString {
			t.Errorf("[%s] Error mismatch. Expected: %q, got %q", tc.name, tc.errString, err.Error())
		}
	}
}

func TestValidateServiceCIDR(t *testing.T) {
	var testCases = []struct {
		name        string
		cidr        string
		expectedStr string
	}{
		{
			name:        "Valid V6",
			cidr:        "fd00:30::/110",
			expectedStr: "",
		},
		{
			name:        "Missing mask",
			cidr:        "fd00:30::",
			expectedStr: "unable to parse test CIDR (fd00:30::)",
		},
		{
			name:        "Missing mask value",
			cidr:        "fd00:30::/",
			expectedStr: "unable to parse test CIDR (fd00:30::/)",
		},
		{
			name:        "Bad IP",
			cidr:        "fd00::30::/110",
			expectedStr: "unable to parse test CIDR (fd00::30::/110)",
		},
		{
			name:        "No CIDR",
			cidr:        "",
			expectedStr: "config missing test CIDR",
		},
		{
			name:        "Valid V4",
			cidr:        "10.96.0.0/12",
			expectedStr: "",
		},
	}
	for _, tc := range testCases {
		err := lazyjack.ValidateCIDR("test", tc.cidr)
		if err == nil {
			if tc.expectedStr != "" {
				t.Errorf("[%s] No error seen, but expected %s", tc.name, tc.expectedStr)
			}
		} else {
			if tc.expectedStr == "" {
				t.Errorf("[%s] Expected no error, but saw: %s", tc.name, err.Error())
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}

func TestValidateNetworkMode(t *testing.T) {
	var testCases = []struct {
		name        string
		mode        string
		expected    string
		expectedStr string
	}{
		{
			name:        "Default is IPv6",
			mode:        "",
			expected:    lazyjack.IPv6NetMode,
			expectedStr: "",
		},
		{
			name:        "IPv4",
			mode:        "ipv4",
			expected:    lazyjack.IPv4NetMode,
			expectedStr: "",
		},
		{
			name:        "IPv6",
			mode:        "ipv6",
			expected:    lazyjack.IPv6NetMode,
			expectedStr: "",
		},
		{
			name:        "dual stack",
			mode:        "dual-stack",
			expected:    lazyjack.DualStackNetMode,
			expectedStr: "",
		},
		{
			name:        "Forced to lowercase",
			mode:        "IPv6",
			expected:    lazyjack.IPv6NetMode,
			expectedStr: "",
		},
		{
			name:        "Invalid mode",
			mode:        "bogus",
			expected:    lazyjack.IPv6NetMode,
			expectedStr: "unsupported network mode \"bogus\" entered",
		},
	}
	for _, tc := range testCases {
		c := &lazyjack.Config{
			General: lazyjack.GeneralSettings{
				Mode: tc.mode,
			},
		}
		err := lazyjack.ValidateNetworkMode(c)
		if err == nil {
			if tc.expectedStr != "" {
				t.Errorf("[%s] No error seen, but expected %s", tc.name, tc.expectedStr)
			} else {
				if c.General.Mode != tc.expected {
					t.Errorf("[%s] Expected mode to be %q, but was %q", tc.name, tc.expected, c.General.Mode)
				}
			}
		} else {
			if tc.expectedStr == "" {
				t.Errorf("[%s] Expected no error, but saw: %s", tc.name, err.Error())
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}

func TestNoModeSpecified(t *testing.T) {
	c := &lazyjack.Config{}
	err := lazyjack.ValidateNetworkMode(c)
	if err != nil {
		t.Errorf("Expected no error, when network mode is not specified")
	}
	if c.General.Mode != "ipv6" {
		t.Errorf("Expected default network mode to be 'ipv6', but was %q", c.General.Mode)
	}
}

func TestValidatePlugin(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Plugin: "bridge",
		},
	}
	err := lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Fatalf("Expected valid plugin selection to work: %s", err.Error())
	}
	if _, ok := c.General.CNIPlugin.(lazyjack.BridgePlugin); !ok {
		t.Fatalf("Expected plugin to be Bridge")
	}
}

func TestValidatePointToPointPlugin(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Plugin: "ptp",
		},
	}
	err := lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Fatalf("Expected valid plugin selection to work: %s", err.Error())
	}
	if _, ok := c.General.CNIPlugin.(lazyjack.PointToPointPlugin); !ok {
		t.Fatalf("Expected plugin to be PTP")
	}
}

func TestFailedInvalidValidatePlugin(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Plugin: "bogus-plugin",
		},
	}
	err := lazyjack.ValidatePlugin(c)
	if err == nil {
		t.Fatalf("Expected invalid  plugin selection to fail: %s", c.General.Plugin)
	}
}

func TestLegacyValidatePlugin(t *testing.T) {
	c := &lazyjack.Config{
		Plugin: "bridge",
	}
	err := lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Fatalf("Expected valid legacy plugin selection to work: %s", err.Error())
	}
	if c.General.Plugin != "bridge" {
		t.Fatalf("Expected legacy plugin to be stored in new field. See %q", c.General.Plugin)
	}
}

func TestFailedInvalidLegacyValidatePlugin(t *testing.T) {
	c := &lazyjack.Config{
		Plugin: "bogus-legacy-plugin",
	}
	err := lazyjack.ValidatePlugin(c)
	if err == nil {
		t.Fatalf("Expected invalid legacy plugin selection to fail: %s", c.Plugin)
	}
}

func TestFailedMissingValidatePlugin(t *testing.T) {
	c := &lazyjack.Config{}
	err := lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Fatalf("Expected config without plugin to work: %s", err.Error())
	}
	if c.General.Plugin != "bridge" {
		t.Fatalf("Expected default plugin to be used. See %q", c.General.Plugin)
	}
}

func TestValidateHost(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				ID: 10,
			},
		},
	}

	err := lazyjack.ValidateHost("master", c)
	if err != nil {
		t.Fatalf("Expected host to be found in config: %s", err.Error())
	}
	err = lazyjack.ValidateHost("no-such-host", c)
	if err == nil {
		t.Fatalf("Expected not to find non-existent host in config")
	}
}

func TestValidateConfigContents(t *testing.T) {
	lazyjack.RegisterExecCommand(func(string, []string) (string, error) { return "v1.11.0", nil })

	cf, err := lazyjack.OpenConfigFile("sample-config.yaml")
	if err != nil {
		t.Fatalf("Test setup failure - unable to open sample config")
	}

	c, err := lazyjack.LoadConfig(cf)
	if err != nil {
		t.Fatalf("Test setup failure - unable to load sample config")
	}
	err = lazyjack.ValidateConfigContents(c, true)
	if err != nil {
		t.Fatalf("Expected to be able to validate sample config: %s", err.Error())
	}
}

func TestNoConfigFileContents(t *testing.T) {
	err := lazyjack.ValidateConfigContents(nil, true)
	if err == nil {
		t.Fatalf("Expected failure, when no config file")
	}
	if err.Error() != "no configuration loaded" {
		t.Fatalf("Expected failure due to no config file, instead, got %q", err.Error())
	}
}

func TestIPv4Test(t *testing.T) {
	isV4 := lazyjack.IsIPv4("10.192.0.0")
	if !isV4 {
		t.Fatalf("Expected true result for IPv4 address")
	}
	isV4 = lazyjack.IsIPv4("2001:db8::")
	if isV4 {
		t.Fatalf("Expected false result for IPv6 address")
	}
}

func TestMakeIPv4Prefix(t *testing.T) {
	var testCases = []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "/8 prefix",
			ip:       "10.0.0.0",
			expected: "10.0.0.",
		},
		{
			name:     "/16 prefix",
			ip:       "10.20.0.0",
			expected: "10.20.0.",
		},
		{
			name:     "/24 prefix",
			ip:       "10.20.30.0",
			expected: "10.20.30.",
		},
		{
			name:     "/12 prefix",
			ip:       "10.224.0.0",
			expected: "10.224.0.",
		},
	}
	for _, tc := range testCases {
		actual := lazyjack.MakeV4PrefixFromNetwork(tc.ip)
		if actual != tc.expected {
			t.Errorf("[%s]: Expected %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestCheckMgmtSize(t *testing.T) {
	var testCases = []struct {
		name     string
		size     int
		expected string
	}{
		{
			name:     "/8 prefix",
			size:     8,
			expected: "",
		},
		{
			name:     "/16 prefix",
			size:     16,
			expected: "",
		},
		{
			name:     "/24 prefix",
			size:     24,
			expected: "only /8 and /16 are supported for an IPv4 management network - have /24",
		},
	}
	for _, tc := range testCases {
		err := lazyjack.CheckMgmtSize(tc.size)
		actual := ""
		if err != nil {
			actual = err.Error()
		}
		if actual != tc.expected {
			t.Errorf("[%s] Expected result %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestCheckPodSize(t *testing.T) {
	var testCases = []struct {
		name     string
		size     int
		expected string
	}{
		{
			name:     "/16 prefix",
			size:     16,
			expected: "",
		},
		{
			name:     "/24 prefix",
			size:     24,
			expected: "only /16 is supported for IPv4 pod networks - have /24",
		},
	}
	for _, tc := range testCases {
		err := lazyjack.CheckPodSize(tc.size)
		actual := ""
		if err != nil {
			actual = err.Error()
		}
		if actual != tc.expected {
			t.Errorf("[%s] Expected result %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestCheckServiceSize(t *testing.T) {
	var testCases = []struct {
		name     string
		size     int
		expected string
	}{
		{
			name:     "/8 prefix",
			size:     8,
			expected: "",
		},
		{
			name:     "/23 prefix",
			size:     23,
			expected: "",
		},
		{
			name:     "/24 prefix",
			size:     24,
			expected: "service subnet size must be /23 or larger - have /24",
		},
	}
	for _, tc := range testCases {
		err := lazyjack.CheckServiceSize(tc.size)
		actual := ""
		if err != nil {
			actual = err.Error()
		}
		if actual != tc.expected {
			t.Errorf("[%s] Expected result %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestCheckUnlimitedSize(t *testing.T) {
	err := lazyjack.CheckUnlimitedSize(80)
	if err != nil {
		t.Errorf("Expected no error on unlimited size check: %s", err.Error())
	}
}

func TestExtractNetInfo(t *testing.T) {
	var testCases = []struct {
		name           string
		cidr           string
		expected       string
		expectedPrefix string
		expectedSize   int
		expectedMode   string
	}{
		{
			name:           "valid v4",
			cidr:           "10.0.0.0/16",
			expected:       "",
			expectedPrefix: "10.0.0.",
			expectedSize:   16,
			expectedMode:   lazyjack.IPv4NetMode,
		},
		{
			name:           "valid v6",
			cidr:           "2001:db8::/64",
			expected:       "",
			expectedPrefix: "2001:db8::",
			expectedSize:   64,
			expectedMode:   lazyjack.IPv6NetMode,
		},
		{
			name:           "no cidr",
			cidr:           "",
			expected:       "missing CIDR",
			expectedPrefix: "",
			expectedSize:   0,
			expectedMode:   lazyjack.IPv6NetMode,
		},
		{
			name:           "bad cidr",
			cidr:           "2001::db8::/64",
			expected:       "invalid CIDR address: 2001::db8::/64",
			expectedPrefix: "",
			expectedSize:   0,
			expectedMode:   lazyjack.IPv6NetMode,
		},
		{
			name:           "bad size",
			cidr:           "10.96.0.0/24",
			expected:       "only /16 is supported for IPv4 pod networks - have /24",
			expectedPrefix: "",
			expectedSize:   0,
			expectedMode:   lazyjack.IPv4NetMode,
		},
	}
	for _, tc := range testCases {
		var actual lazyjack.NetInfo
		err := lazyjack.ExtractNetInfo(tc.cidr, &actual, lazyjack.CheckPodSize)
		actualErr := ""
		if err != nil {
			actualErr = err.Error()
		}
		if actualErr != tc.expected {
			t.Errorf("[%s] expected result %q, got %q", tc.name, tc.expected, actualErr)
		} else if actualErr == "" {
			if actual.Prefix != tc.expectedPrefix {
				t.Errorf("[%s] expected prefix %q, got %q", tc.name, tc.expectedPrefix, actual.Prefix)
			} else if actual.Size != tc.expectedSize {
				t.Errorf("[%s] expected size %d, got %d", tc.name, tc.expectedSize, actual.Size)
			} else if actual.Mode != tc.expectedMode {
				t.Errorf("[%s] expected size %q, got %q", tc.name, tc.expectedMode, actual.Mode)
			}
		}
	}
}

func TestValidatePodMTU(t *testing.T) {
	var testCases = []struct {
		name        string
		mtu         int
		expectedMTU int
		expectedStr string
	}{
		{
			name:        "Default 1500",
			mtu:         0,
			expectedMTU: 1500,
			expectedStr: "",
		},
		{
			name:        "Overridden MTU",
			mtu:         9000,
			expectedMTU: 9000,
			expectedStr: "",
		},
		{
			name:        "Overridden with minium MTU",
			mtu:         1280,
			expectedMTU: 1280,
			expectedStr: "",
		},
		{
			name:        "Too small",
			mtu:         1279,
			expectedMTU: 0,
			expectedStr: "MTU (1279) is less than minimum MTU for IPv6 (1280)",
		},
	}
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			MTU: -1,
		},
	}
	for _, tc := range testCases {
		c.Pod.MTU = tc.mtu
		err := lazyjack.ValidatePodFields(c)
		if err == nil {
			if tc.expectedStr != "" {
				t.Errorf("[%s] No error seen, but expected %s", tc.name, tc.expectedStr)
			} else if c.Pod.MTU != tc.expectedMTU {
				t.Errorf("[%s] Expected MTU %d, got %d", tc.name, tc.expectedMTU, c.Pod.MTU)
			}
		} else {
			if tc.expectedStr == "" {
				t.Errorf("[%s] Expected no error, but saw: %s", tc.name, err.Error())
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}

func TestDeprecatedAAAASupport(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			AllowIPv6Use: true, // Deprecated field
		},
	}

	err := lazyjack.ValidateDNS64Fields(c)
	if err != nil {
		t.Fatalf("Expected validating DNS64 fields to succeed, but see error: %s", err.Error())
	}
	if !c.DNS64.AllowAAAAUse {
		t.Fatalf("Expected allow AAAA use field to be set by deprecated value")
	}
}

func TestIPv4Mapping(t *testing.T) {
	var testCases = []struct {
		name        string
		support_net string
		v4IP        string
		v4Pool      string
		expectedStr string
	}{
		{
			name:        "Mapping pool is in subnet",
			support_net: "172.18.0.0/16",
			v4IP:        "172.18.0.200",
			v4Pool:      "172.18.0.128/25",
			expectedStr: "",
		},
		{
			name:        "Bad mapping IP",
			support_net: "172.18.0.0/16",
			v4IP:        "172.18.0.0.200",
			v4Pool:      "172.18.0.128/25",
			expectedStr: "v4 mapping IP (172.18.0.0.200) is invalid",
		},
		{
			name:        "Bad mapping CIDR",
			support_net: "172.18.0.0/16",
			v4IP:        "172.18.0.200",
			v4Pool:      "172.18.0.128/500",
			expectedStr: "v4 mapping CIDR (172.18.0.128/500) is invalid: invalid CIDR address: 172.18.0.128/500",
		},
		{
			name:        "Bad support net CIDR",
			support_net: "172.18.0.0/400",
			v4IP:        "172.18.0.200",
			v4Pool:      "172.18.0.128/500",
			expectedStr: "v4 support network (172.18.0.0/400) is invalid: invalid CIDR address: 172.18.0.0/400",
		},
		{
			name:        "Missing mapping IP",
			support_net: "172.18.0.0/16",
			v4IP:        "",
			v4Pool:      "172.18.0.128/25",
			expectedStr: "missing IPv4 mapping IP",
		},
		{
			name:        "Missing mapping CIDR",
			support_net: "172.18.0.0/16",
			v4IP:        "172.18.0.200",
			v4Pool:      "",
			expectedStr: "missing IPv4 mapping CIDR",
		},
		{
			name:        "Missing support net CIDR",
			support_net: "",
			v4IP:        "",
			v4Pool:      "172.18.0.128/25",
			expectedStr: "missing IPv4 support network CIDR",
		},
		{
			name:        "Mapping CIDR not in subnet",
			support_net: "172.18.0.0/16",
			v4IP:        "172.18.0.200",
			v4Pool:      "172.22.0.128/25",
			expectedStr: "V4 mapping CIDR (172.22.0.128/25) is not within IPv4 support subnet (172.18.0.0/16)",
		},
		{
			name:        "Mapping IP not in support net",
			support_net: "172.18.0.0/16",
			v4IP:        "172.22.0.200",
			v4Pool:      "172.22.0.128/25",
			expectedStr: "V4 mapping IP (172.22.0.200) is not within IPv4 support subnet (172.18.0.0/16)",
		},
	}

	for _, tc := range testCases {
		c := &lazyjack.Config{
			NAT64: lazyjack.NAT64Config{
				V4MappingCIDR: tc.v4Pool,
				V4MappingIP:   tc.v4IP,
			},
			Support: lazyjack.SupportNetwork{
				V4CIDR: tc.support_net,
			},
			General: lazyjack.GeneralSettings{
				Mode: lazyjack.IPv6NetMode,
			},
		}
		err := lazyjack.ValidateNAT64Fields(c)
		if err == nil {
			if tc.expectedStr != "" {
				t.Errorf("[%s] No error seen, but expected %s", tc.name, tc.expectedStr)
			}
		} else {
			if tc.expectedStr == "" {
				t.Errorf("[%s] Expected no error, but saw: %s", tc.name, err.Error())
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}

func TestSkipValidateNAT64ForV4(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv4NetMode,
		},
	}
	err := lazyjack.ValidateNAT64Fields(c)
	if err != nil {
		t.Fatalf("Expected validation of NAT64 to be skipped for IPv4: %s", err.Error())
	}
}

func TestCalculateDerivedFieldsSuccess(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err != nil {
		t.Fatalf("Expected derived fields parsed OK, but see error: %s", err.Error())
	}
	expectedMgmtPrefix := "fd00:20::"
	if c.Mgmt.Info[0].Prefix != expectedMgmtPrefix {
		t.Errorf("Derived management prefix is incorrect. Expected %q, got %q", expectedMgmtPrefix, c.Mgmt.Info[0].Prefix)
	}
	expectedMgmtSize := 64
	if c.Mgmt.Info[0].Size != expectedMgmtSize {
		t.Errorf("Derived management size is incorrect. Expected %d, got %d", expectedMgmtSize, c.Mgmt.Info[0].Size)
	}
	expectedServicePrefix := "fd00:30::"
	if c.Service.Info.Prefix != expectedServicePrefix {
		t.Errorf("Derived service prefix is incorrect. Expected %q, got %q", expectedServicePrefix, c.Service.Info.Prefix)
	}
	expectedSupportPrefix := "fd00:10::"
	if c.Support.Info.Prefix != expectedSupportPrefix {
		t.Errorf("Derived support prefix is incorrect. Expected %q, got %q", expectedSupportPrefix, c.Support.Info.Prefix)
	}
	expectedSupportSize := 64
	if c.Support.Info.Size != expectedSupportSize {
		t.Errorf("Derived support size is incorrect. Expected %d, got %d", expectedSupportSize, c.Support.Info.Size)
	}
	expectedPodPrefix := "fd00:40:0:0:"
	if c.Pod.Info[0].Prefix != expectedPodPrefix {
		t.Errorf("Derived pod prefix is incorrect. Expected %q, got %q", expectedPodPrefix, c.Pod.Info[0].Prefix)
	}
	expectedPodSize := 80
	if c.Pod.Info[0].Size != expectedPodSize {
		t.Errorf("Derived pod size is incorrect. Expected %d, got %d", expectedPodSize, c.Pod.Info[0].Size)
	}
	expectedDNS64Prefix := "fd00:10:64:ff9b::"
	if c.DNS64.CIDRPrefix != expectedDNS64Prefix {
		t.Errorf("Derived DNS64 prefix is incorrect. Expected %q, got %q", expectedDNS64Prefix, c.DNS64.CIDRPrefix)
	}
}

func TestCalculateDerivedFieldsDualStackSuccess(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err != nil {
		t.Fatalf("Expected derived fields parsed OK, but see error: %s", err.Error())
	}
	expectedMgmtPrefix := "fd00:20::"
	if c.Mgmt.Info[0].Prefix != expectedMgmtPrefix {
		t.Errorf("Derived management prefix is incorrect. Expected %q, got %q", expectedMgmtPrefix, c.Mgmt.Info[0].Prefix)
	}
	expectedMgmtSize := 64
	if c.Mgmt.Info[0].Size != expectedMgmtSize {
		t.Errorf("Derived management size is incorrect. Expected %d, got %d", expectedMgmtSize, c.Mgmt.Info[0].Size)
	}
	expectedMgmtPrefix = "10.192.0."
	if c.Mgmt.Info[1].Prefix != expectedMgmtPrefix {
		t.Errorf("Derived management prefix2 is incorrect. Expected %q, got %q", expectedMgmtPrefix, c.Mgmt.Info[1].Prefix)
	}
	expectedMgmtSize = 16
	if c.Mgmt.Info[1].Size != expectedMgmtSize {
		t.Errorf("Derived management size2 is incorrect. Expected %d, got %d", expectedMgmtSize, c.Mgmt.Info[1].Size)
	}
	expectedServicePrefix := "fd00:30::"
	if c.Service.Info.Prefix != expectedServicePrefix {
		t.Errorf("Derived service prefix is incorrect. Expected %q, got %q", expectedServicePrefix, c.Service.Info.Prefix)
	}
	expectedPodPrefix := "fd00:40:0:0:"
	if c.Pod.Info[0].Prefix != expectedPodPrefix {
		t.Errorf("Derived pod prefix is incorrect. Expected %q, got %q", expectedPodPrefix, c.Pod.Info[0].Prefix)
	}
	expectedPodSize := 80
	if c.Pod.Info[0].Size != expectedPodSize {
		t.Errorf("Derived pod size is incorrect. Expected %d, got %d", expectedPodSize, c.Pod.Info[0].Size)
	}
	expectedPodPrefix = "10.244.0."
	if c.Pod.Info[1].Prefix != expectedPodPrefix {
		t.Errorf("Derived pod prefix2 is incorrect. Expected %q, got %q", expectedPodPrefix, c.Pod.Info[1].Prefix)
	}
	expectedPodSize = 24
	if c.Pod.Info[1].Size != expectedPodSize {
		t.Errorf("Derived pod size2 is incorrect. Expected %d, got %d", expectedPodSize, c.Pod.Info[1].Size)
	}
}

func TestFailedMgmtCIDRCalculateDerivedFields(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00::20::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid management CIDR")
	}
	expectedMsg := "invalid management network: invalid CIDR address: fd00::20::/64"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedV4MgmtCIDRMissingCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second management V4 CIDR")
	}
	expectedMsg := "dual-stack mode management network only has ipv6 CIDR, need ipv4 CIDR"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedV6MgmtCIDRMissingCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "10.192.0.0/16",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second management V6 CIDR")
	}
	expectedMsg := "dual-stack mode management network only has ipv4 CIDR, need ipv6 CIDR"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedSecondMgmtCIDRBadCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0.0/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid second management CIDR")
	}
	expectedMsg := "invalid management network CIDR2: invalid CIDR address: 10.192.0.0.0/64"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedMgmtCIDRBothV6CalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "fd00:50::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second management V4 CIDR")
	}
	expectedMsg := "for dual-stack both management networks specified are ipv6 mode - need ipv4 info"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedMgmtCIDRBothV4CalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "10.192.0.0/16",
			CIDR2: "10.193.0.0/16",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second management V6 CIDR")
	}
	expectedMsg := "for dual-stack both management networks specified are ipv4 mode - need ipv6 info"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedTwoCIDRSCalculateDerivedFieldsForIPv6(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "10.192.0.0/16",
			CIDR2: "fd00:20::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second management CIDR")
	}
	expectedMsg := "see second management network CIDR (10.192.0.0/16, fd00:20::/64), when in ipv6 mode"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedServiceCIDRCalculateDerivedFields(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00::30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid service CIDR")
	}
	expectedMsg := "invalid service network: invalid CIDR address: fd00::30::/110"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedSupportCIDRCalculateDerivedFields(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/6a4",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid support CIDR")
	}
	expectedMsg := "invalid support network: invalid CIDR address: fd00:10::/6a4"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedDNS64CalculateDerivedFields(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid DNS64 CIDR")
	}
	expectedMsg := "invalid DNS64 CIDR: invalid CIDR address: fd00:10:64:ff9b::96"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedBadPodCIDRCalculateDerivedFields(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00::40::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid pod CIDR")
	}
	expectedMsg := "invalid pod network: invalid CIDR address: fd00::40::/72"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedBadSecondPodCIDRCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0.0/35",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with invalid second pod CIDR")
	}
	expectedMsg := "invalid pod network CIDR2: invalid CIDR address: 10.244.0.0.0/35"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedMissingV4PodCIDRCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "fd00:40::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second V4 pod CIDR")
	}
	expectedMsg := "dual-stack mode pod network only has ipv6 CIDR, need ipv4 CIDR"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedMissingV6PodCIDRCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with missing second V6 pod CIDR")
	}
	expectedMsg := "dual-stack mode pod network only has ipv4 CIDR, need ipv6 CIDR"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedBothV6PodCIDRCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "fd00:50::/72",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with two IPv6 pod CIDRs")
	}
	expectedMsg := "for dual-stack both pod networks specified are ipv6 mode - need ipv4 info"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedBothV4PodCIDRCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "10.244.0.0/16",
			CIDR2: "10.245.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with two IPv4 pod CIDRs")
	}
	expectedMsg := "for dual-stack both pod networks specified are ipv4 mode - need ipv6 info"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedSecondPodCIDRCalculateDerivedFieldsV6Mode(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.IPv6NetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with second pod CIDR in IPv6 mode")
	}
	expectedMsg := "see second pod network CIDR (fd00:40::/72, 10.244.0.0/16), when in ipv6 mode"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestFailedSupportCIDRCalculateDerivedFieldsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR:  "fd00:20::/64",
			CIDR2: "10.192.0.0/16",
		},
		Service: lazyjack.ServiceNetwork{
			CIDR: "fd00:30::/110",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
		Pod: lazyjack.PodNetwork{
			CIDR:  "fd00:40::/72",
			CIDR2: "10.244.0.0/16",
		},
		General: lazyjack.GeneralSettings{
			Mode: lazyjack.DualStackNetMode,
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Fatalf("Expected failure with support CIDR in dual-stack mode")
	}
	expectedMsg := "support CIDR (fd00:10::/64) is unsupported in dual-stack mode"
	if err.Error() != expectedMsg {
		t.Fatalf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestMakePrefixFromNetwork(t *testing.T) {
	var testCases = []struct {
		name     string
		network  string
		size     int
		expected string
	}{
		{
			name:     "padded with zeros",
			network:  "fd00:40::",
			size:     72,
			expected: "fd00:40:0:0:",
		},
		{
			name:     "multiple of 16 padded with zeros",
			network:  "fd00:40::",
			size:     64,
			expected: "fd00:40:0:0:",
		},
		{
			name:     "no padding needed",
			network:  "fd00:10:20:30:40::",
			size:     80,
			expected: "fd00:10:20:30:40:",
		},
		{
			name:     "trimmed low byte",
			network:  "fd00:10:20:30:40:5000::",
			size:     88,
			expected: "fd00:10:20:30:40:50",
		},
		{
			name:     "pad to next colon",
			network:  "fd00:10:20:30::",
			size:     80,
			expected: "fd00:10:20:30:0:",
		},
	}
	for _, tc := range testCases {
		actual := lazyjack.MakePrefixFromNetwork(tc.network, tc.size)
		if actual != tc.expected {
			t.Errorf("[%s] Expected: %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestValidateVersionsCommand(t *testing.T) {
	var testCases = []struct {
		name        string
		input       string
		version     string
		expectedStr string
	}{
		{
			name:        "no version info",
			input:       "",
			version:     "",
			expectedStr: "Unable to parse Kubeadm version from \"\"",
		},
		{
			name:        "good version",
			input:       "v1.10.8",
			version:     "1.10",
			expectedStr: "",
		},
		{
			name:        "built version",
			input:       "v1.13.0-dirty",
			version:     "1.13",
			expectedStr: "",
		},
		{
			name:        "malformed version",
			input:       "1.10.8",
			version:     "",
			expectedStr: "Unable to parse Kubeadm version from \"1.10.8\"",
		},
		{
			name:        "invalid version",
			input:       "1.10",
			version:     "",
			expectedStr: "Unable to parse Kubeadm version from \"1.10\"",
		},
	}
	for _, tc := range testCases {
		version, err := lazyjack.ParseVersion(tc.input)
		if tc.expectedStr == "" {
			if err != nil {
				t.Errorf("[%s] Did not expect error, but see %q", tc.name, err.Error())
			} else if tc.version != version {
				t.Errorf("[%s] Expected version %q, but got %q", tc.name, tc.version, version)
			}
		} else {
			if err == nil {
				t.Errorf("[%s] Expected error %q, but no error seen", tc.name, tc.expectedStr)
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, but got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}

func TestValidateSoftwareVersions(t *testing.T) {
	var testCases = []struct {
		name        string
		execCmd     lazyjack.ExecCommandFuncType
		version     string
		k8sVersion  string
		expectedStr string
	}{
		{
			name: "kubeadm ok",
			execCmd: func(string, []string) (string, error) {
				return "v1.12.33", nil
			},
			version:     "1.12",
			k8sVersion:  "",
			expectedStr: "",
		},
		{
			name: "local version",
			execCmd: func(string, []string) (string, error) {
				return "v1.13.0-alpha.0-dirty", nil
			},
			version:     "1.13",
			k8sVersion:  "",
			expectedStr: "",
		},
		{
			name: "unabe to get version",
			execCmd: func(string, []string) (string, error) {
				return "", fmt.Errorf("kubeadm: command not found")
			},
			version:     "",
			k8sVersion:  "",
			expectedStr: "Unable to get version of KubeAdm: kubeadm: command not found",
		},
		{
			name: "unabe to parse version",
			execCmd: func(string, []string) (string, error) {
				return "bad-version-output", nil
			},
			version:     "",
			k8sVersion:  "",
			expectedStr: "Unable to parse Kubeadm version from \"bad-version-output\"",
		},
		{
			name: "valid k8s verison",
			execCmd: func(string, []string) (string, error) {
				return "v1.12.33", nil
			},
			version:     "1.12",
			k8sVersion:  "v1.12.0",
			expectedStr: "",
		},
		{
			name: "latest k8s verison",
			execCmd: func(string, []string) (string, error) {
				return "v1.12.33", nil
			},
			version:     "1.12",
			k8sVersion:  "latest",
			expectedStr: "",
		},
		{
			name: "unsupported verison (ok)",
			execCmd: func(string, []string) (string, error) {
				return "v1.9.1", nil
			},
			version:     "1.9",
			k8sVersion:  "v1.9.3",
			expectedStr: "",
		},
		{
			name: "k8s mismatch",
			execCmd: func(string, []string) (string, error) {
				return "v1.12.33", nil
			},
			version:     "1.12",
			k8sVersion:  "v1.11.1",
			expectedStr: "specified Kubernetes verson (\"v1.11.1\") does not match KubeAdm version (1.12)",
		},
		{
			name: "k8s not full qual",
			execCmd: func(string, []string) (string, error) {
				return "v1.12.33", nil
			},
			version:     "1.12",
			k8sVersion:  "v1.12",
			expectedStr: "unable to parse Kubernetes version specified (\"v1.12\"): Unable to parse Kubeadm version from \"v1.12\"",
		},
		{
			name: "k8s invalid",
			execCmd: func(string, []string) (string, error) {
				return "v1.12.33", nil
			},
			version:     "1.12",
			k8sVersion:  "1.12.0",
			expectedStr: "unable to parse Kubernetes version specified (\"1.12.0\"): Unable to parse Kubeadm version from \"1.12.0\"",
		},
	}

	for _, tc := range testCases {
		lazyjack.RegisterExecCommand(tc.execCmd)
		c := &lazyjack.Config{
			General: lazyjack.GeneralSettings{
				K8sVersion: tc.k8sVersion,
			},
		}
		err := lazyjack.ValidateSoftwareVersions(c)
		if tc.expectedStr == "" {
			if err != nil {
				t.Errorf("[%s] Did not expect error, but see %q", tc.name, err.Error())
			} else if tc.version != c.General.KubeAdmVersion {
				t.Errorf("[%s] Expected version %q, but got %q", tc.name, tc.version, c.General.KubeAdmVersion)
			}
		} else {
			if err == nil {
				t.Errorf("[%s] Expected error %q, but no error seen", tc.name, tc.expectedStr)
			} else if err.Error() != tc.expectedStr {
				t.Errorf("[%s] Expected error %q, but got %q", tc.name, tc.expectedStr, err.Error())
			}
		}
	}
}
