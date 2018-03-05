package lazyjack_test

import (
	"bytes"
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
			expectedStr: "Missing command",
		},
		{
			name:        "Unknown command",
			command:     "foo",
			expected:    "",
			expectedStr: "Unknown command \"foo\"",
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
		t.Errorf("Did not expect to have config, with empty filename")
	}
	if err.Error() != "Unable to open config file \"\": open : no such file or directory" {
		t.Errorf("Expected error message, when trying to open empty filename")
	}

	// File does not exist
	cf, err = lazyjack.OpenConfigFile("non-existing-file")
	if cf != nil {
		t.Errorf("Did not expect to have config, with non-existing filename")
	}
	if err.Error() != "Unable to open config file \"non-existing-file\": open non-existing-file: no such file or directory" {
		t.Errorf("Expected error message, when trying to open non-existing filename")
	}

	// Have a file
	cf, err = lazyjack.OpenConfigFile("sample-config.yaml")
	if cf == nil {
		t.Errorf("Expected to open sample configuration file")
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

func TestLoadConfig(t *testing.T) {
	// Malformed config file

	badYAML := `# Simple YAML file with (invalid) tab character
topology:
    good-host:
       interface: "eth0"
\topmodes: "master"
       id: 2`
	stream := &ClosingBuffer{bytes.NewBufferString(badYAML)}
	config, err := lazyjack.LoadConfig(stream)
	if config != nil {
		t.Errorf("Should not have config loaded, if YAML is malformed")
	}
	if err.Error() != "Failed to parse config: yaml: line 5: did not find expected key" {
		t.Errorf("Error message is not correct for malformed YAML file (%s)", err.Error())
	}

	// Good config file
	goodYAML := `# Valid YAML file
plugin: bridge
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
    ip: "fd00:10::100"`

	stream = &ClosingBuffer{bytes.NewBufferString(goodYAML)}
	config, err = lazyjack.LoadConfig(stream)

	if err != nil {
		t.Errorf("Unexpected error, when reading config")
	}
	if config == nil {
		t.Errorf("Should have a config")
	}

	node, ok := config.Topology["my-master"]
	if !ok {
		t.Errorf("Expected to have configuration for my-master node")
	}
	if node.Interface != "eth0" || node.ID != 2 || node.OperatingModes != "master dns64 nat64" {
		t.Errorf("Incorrect config for node my-master (%+v)", node)
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
		config.DNS64.ServerIP != "fd00:10::100" {
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
		t.Errorf("Expected failure with duplicate IDs")
	}
	// Order of node names is not guaranteed, so just check first part of msg
	if !strings.HasPrefix(err.Error(), "Duplicate node ID 10 seen for node") {
		t.Errorf("Error message is not correct (%s)", err.Error())
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
		t.Errorf("Should not be an error, as all IDs are unique")
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
			expectedStr: "Missing operating mode for \"test-node\"",
		},
		{
			name:        "No modes specified",
			opMode:      "   ",
			expectedStr: "Missing operating mode for \"test-node\"",
		},
		{
			name:        "Unknown mode",
			opMode:      "monster",
			expectedStr: "Invalid operating mode \"monster\" for \"test-node\"",
		},
		{
			name:        "Master and minion",
			opMode:      "minion master",
			expectedStr: "Invalid combination of modes for \"test-node\"",
		},
		// Don't currently support just DNS or just NAT
		// TODO: Decide if should allow DNS only/NAT only
		{
			name:        "Missing DNS64",
			opMode:      "nat64",
			expectedStr: "Missing \"dns64\" mode for \"test-node\"",
		},
		{
			name:        "Missing NAT64",
			opMode:      "dns64",
			expectedStr: "Missing \"nat64\" mode for \"test-node\"",
		},
	}

	for _, tc := range testCases {
		node := &lazyjack.Node{Name: "test-node", OperatingModes: tc.opMode}
		err := lazyjack.ValidateNodeOpModes(node)
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
	}

	err := lazyjack.ValidateOpModesForAllNodes(c)
	if err == nil {
		t.Errorf("Expected to see error, when configuration has duplicate master nodes")
	} else if err.Error() != "Found multiple nodes with \"master\" operating mode" {
		t.Errorf("Duplicate master nodes error message wrong (%s)", err.Error())
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
	}

	err := lazyjack.ValidateOpModesForAllNodes(c)
	if err == nil {
		t.Errorf("Expected to see error, when configuration has no master node entry")
	} else if err.Error() != "No master node configuration" {
		t.Errorf("No master node error message wrong (%s)", err.Error())
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
			errString: "Missing token in config file",
		},
		{
			name:      "Wrong length",
			input:     "7aee33.05f81856d78346b",
			errString: "Invalid token length (22)",
		},
		{
			name:      "Invalid value",
			input:     "ABCDEF.hasbadcharacters",
			errString: "Token is invalid \"ABCDEF.hasbadcharacters\"",
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
			errString: "Missing token certificate hash in config file",
		},
		{
			name:      "Wrong length",
			input:     "123456789012345678901234567890123456789012345678901234567890abc",
			errString: "Invalid token certificate hash length (63)",
		},
		{
			name:      "Invalid value",
			input:     "123456789012345678901234567890123456789012345678hasbadcharacters",
			errString: "Token certificate hash is invalid \"123456789012345678901234567890123456789012345678hasbadcharacters\"",
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
			name:        "Valid",
			cidr:        "fd00:30::/110",
			expectedStr: "",
		},
		{
			name:        "Missing mask",
			cidr:        "fd00:30::",
			expectedStr: "Unable to parse test CIDR (fd00:30::)",
		},
		{
			name:        "Missing mask value",
			cidr:        "fd00:30::/",
			expectedStr: "Unable to parse test CIDR (fd00:30::/)",
		},
		{
			name:        "Bad IP",
			cidr:        "fd00::30::/110",
			expectedStr: "Unable to parse test CIDR (fd00::30::/110)",
		},
		{
			name:        "No CIDR",
			cidr:        "",
			expectedStr: "Config missing test CIDR",
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

func TestValidatePlugin(t *testing.T) {
	// Try valid plugin
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Plugin: "bridge",
		},
	}
	err := lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Errorf("Expected valid plugin selection to work: %s", err.Error())
	}

	// Try invalid plugin
	c = &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Plugin: "bogus-plugin",
		},
	}
	err = lazyjack.ValidatePlugin(c)
	if err == nil {
		t.Errorf("Expected invalid  plugin selection to fail: %s", c.General.Plugin)
	}

	// Try legacy plugin
	c = &lazyjack.Config{
		Plugin: "bridge",
	}
	err = lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Errorf("Expected valid legacy plugin selection to work: %s", err.Error())
	}
	if c.General.Plugin != "bridge" {
		t.Errorf("Expected legacy plugin to be stored in new field. See %q", c.General.Plugin)
	}

	// Try invalid legacy plugin
	c = &lazyjack.Config{
		Plugin: "bogus-legacy-plugin",
	}
	err = lazyjack.ValidatePlugin(c)
	if err == nil {
		t.Errorf("Expected invalid legacy plugin selection to fail: %s", c.Plugin)
	}

	// Try no plugin specified
	c = &lazyjack.Config{}
	err = lazyjack.ValidatePlugin(c)
	if err != nil {
		t.Errorf("Expected config without plugin to work: %s", err.Error())
	}
	if c.General.Plugin != "bridge" {
		t.Errorf("Expected default plugin to be used. See %q", c.General.Plugin)
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
		t.Errorf("Expected host to be found in config: %s", err.Error())
	}
	err = lazyjack.ValidateHost("no-such-host", c)
	if err == nil {
		t.Errorf("Expected not to find non-existent host in config")
	}
}

func TestValidateConfigContents(t *testing.T) {
	cf, err := lazyjack.OpenConfigFile("sample-config.yaml")
	if err != nil {
		t.Errorf("Test setup failure - unable to open sample config")
	}

	c, err := lazyjack.LoadConfig(cf)
	if err != nil {
		t.Errorf("Test setup failure - unable to load sample config")
	}
	err = lazyjack.ValidateConfigContents(c, true)
	if err != nil {
		t.Errorf("Expected to be able to validate sample config: %s", err.Error())
	}
}

func TestNoConfigFileContents(t *testing.T) {
	err := lazyjack.ValidateConfigContents(nil, true)
	if err == nil {
		t.Errorf("Expected failure, when no config file")
	}
	if err.Error() != "No configuration loaded" {
		t.Errorf("Expected failure due to no config file, instead, got %q", err.Error())
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
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err != nil {
		t.Errorf("Expected derived fields parsed OK, but see error: %s", err.Error())
	}
	expectedMgmtPrefix := "fd00:20::"
	if c.Mgmt.Prefix != expectedMgmtPrefix {
		t.Errorf("Derived management prefix is incorrect. Expected %q, got %q", expectedMgmtPrefix, c.Mgmt.Prefix)
	}
	expectedMgmtSize := 64
	if c.Mgmt.Size != expectedMgmtSize {
		t.Errorf("Derived management size is incorrect. Expected %d, got %d", expectedMgmtSize, c.Mgmt.Size)
	}
	expectedSupportPrefix := "fd00:10::"
	if c.Support.Prefix != expectedSupportPrefix {
		t.Errorf("Derived support prefix is incorrect. Expected %q, got %q", expectedSupportPrefix, c.Support.Prefix)
	}
	expectedSupportSize := 64
	if c.Support.Size != expectedSupportSize {
		t.Errorf("Derived support size is incorrect. Expected %d, got %d", expectedSupportSize, c.Support.Size)
	}
	expectedDNS64Prefix := "fd00:10:64:ff9b::"
	if c.DNS64.CIDRPrefix != expectedDNS64Prefix {
		t.Errorf("Derived support prefix is incorrect. Expected %q, got %q", expectedDNS64Prefix, c.DNS64.CIDRPrefix)
	}
}

func TestCalculateDerivedFieldsFailures(t *testing.T) {
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00::20::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
	}

	err := lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Errorf("Expected failure with invalid management CIDR")
	}
	expectedMsg := "Invalid management network CIDR: invalid CIDR address: fd00::20::/64"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}

	c = &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/6a4",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::/96",
		},
	}

	err = lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Errorf("Expected failure with invalid support CIDR")
	}
	expectedMsg = "Invalid support network CIDR: invalid CIDR address: fd00:10::/6a4"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}

	c = &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			CIDR: "fd00:20::/64",
		},
		Support: lazyjack.SupportNetwork{
			CIDR: "fd00:10::/64",
		},
		DNS64: lazyjack.DNS64Config{
			CIDR: "fd00:10:64:ff9b::96",
		},
	}

	err = lazyjack.CalculateDerivedFields(c)
	if err == nil {
		t.Errorf("Expected failure with invalid DNS64 CIDR")
	}
	expectedMsg = "Invalid DNS64 CIDR: invalid CIDR address: fd00:10:64:ff9b::96"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}
