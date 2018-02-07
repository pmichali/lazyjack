package orca_test

import (
	"bytes"
	"orca"
	"testing"
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
		command, err := orca.ValidateCommand(tc.command)
		if command != tc.expected {
			t.Errorf("FAILED: [%s] Expected %s, got %s", tc.name, tc.expected, command)
		}
		if tc.expectedStr != "" {
			actualErrStr := err.Error()
			if actualErrStr != tc.expectedStr {
				t.Errorf("FAILED: [%s] Expected error string %q, got %q", tc.name, tc.expectedStr, actualErrStr)
			}
		}
	}
}

func TestValidateConfig(t *testing.T) {
	// No file specified
	cf, err := orca.ValidateConfigFile("")
	if cf != nil {
		t.Errorf("Did not expect to have config, with empty filename")
	}
	if err.Error() != "Unable to open config file \"\": open : no such file or directory" {
		t.Errorf("Expected error message, when trying to open empty filename")
	}

	// File does not exist
	cf, err = orca.ValidateConfigFile("non-existing-file")
	if cf != nil {
		t.Errorf("Did not expect to have config, with non-existing filename")
	}
	if err.Error() != "Unable to open config file \"non-existing-file\": open non-existing-file: no such file or directory" {
		t.Errorf("Expected error message, when trying to open non-existing filename")
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
	config, err := orca.LoadConfig(stream)
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
    subnet: "fd00:10::"
    size: 64
mgmt_net:
    subnet: "fd00:20::"
    size: 64
nat64:
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
    ip: "fd00:10::200"
dns64:
    remote_server: "8.8.8.8"  # Could be a internal/company DNS server
    prefix: "fd00:10:64:ff9b::"
    prefix_size: 96
    ip: "fd00:10::100"`

	stream = &ClosingBuffer{bytes.NewBufferString(goodYAML)}
	config, err = orca.LoadConfig(stream)

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

	if config.Support.Subnet != "fd00:10::" || config.Support.Size != 64 {
		t.Errorf("Support net config parse failed (%+v)", config.Support)
	}

	if config.Mgmt.Subnet != "fd00:20::" || config.Mgmt.Size != 64 {
		t.Errorf("Management net config parse failed (%+v)", config.Mgmt)
	}

	if config.NAT64.V4MappingCIDR != "172.18.0.128/25" ||
		config.NAT64.V4MappingIP != "172.18.0.200" ||
		config.NAT64.ServerIP != "fd00:10::200" {
		t.Errorf("NAT64 config parse failure (%+v)", config.NAT64)
	}

	if config.DNS64.RemoteV4Server != "8.8.8.8" ||
		config.DNS64.Prefix != "fd00:10:64:ff9b::" ||
		config.DNS64.PrefixSize != 96 ||
		config.DNS64.ServerIP != "fd00:10::100" {
		t.Errorf("DNS64 config parse failure (%+v)", config.DNS64)
	}
}
