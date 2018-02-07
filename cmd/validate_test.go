package orca_test

import (
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

func TestLoadConfig(t *testing.T) {
	config, err := orca.ValidateAndLoadConfig("")
	if config != nil {
		t.Errorf("Did not expect to have config, with empty filename")
	}
	if err.Error() != "Unable to open config file \"\": open : no such file or directory" {
		t.Errorf("Expected error message, when trying to open empty filename")
	}

	config, err = orca.ValidateAndLoadConfig("non-existing-file")
	if config != nil {
		t.Errorf("Did not expect to have config, with non-existing filename")
	}
	if err.Error() != "Unable to open config file \"non-existing-file\": open non-existing-file: no such file or directory" {
		t.Errorf("Expected error message, when trying to open non-existing filename")
	}
}
