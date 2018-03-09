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

func TestCreateCertKeyArea(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	err := lazyjack.CreateCertKeyArea(basePath)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create area %q: %s", basePath, err.Error())
	}
	HelperCleanupArea(basePath, t)
}

func TestFailureToCreateCertKeyArea(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")

	// Make it not readable, so that it cannot be removed
	err := os.MkdirAll(basePath, 0400)
	if err != nil {
		t.Errorf("ERROR: Test setup failure: %s", err.Error())
	}
	defer HelperCleanupArea(basePath, t)

	err = lazyjack.CreateCertKeyArea(basePath)
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to clear out old %q as part of creating area", basePath)
	}
}

func TestBuildArgsForCAKey(t *testing.T) {
	args := lazyjack.BuildArgsForCAKey(lazyjack.WorkArea)
	actual := strings.Join(args, " ")
	expected := "genrsa -out /tmp/lazyjack/certs/ca.key 2048"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestFailingCreateKeyForCA(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)
	// Path does not exist to store key
	err := lazyjack.CreateKeyForCA(basePath)
	if err == nil {
		t.Errorf("FAILED: Expected that CA key could not be created")
	}
}

func TestBuildArgsForCACert(t *testing.T) {
	args := lazyjack.BuildArgsForCACert("fd00:100::", 2, lazyjack.WorkArea)
	actual := strings.Join(args, " ")
	expected := "req -x509 -new -nodes -key /tmp/lazyjack/certs/ca.key -subj /CN=fd00:100::2 -days 10000 -out /tmp/lazyjack/certs/ca.crt"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestFailingCreateCertificateForCA(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)
	// No input key and no area to write certificate
	err := lazyjack.CreateCertificateForCA("fd00:100::", 2, basePath)
	if err == nil {
		t.Errorf("FAILED: Expected that CA cert could not be created")
	}
}

func TestBuildArgsForX509Cert(t *testing.T) {
	args := lazyjack.BuildArgsForX509Cert(lazyjack.WorkArea)
	actual := strings.Join(args, " ")
	expected := "x509 -pubkey -in /tmp/lazyjack/certs/ca.crt"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestFailCreateX509CertForCA(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)
	// No input file and no area to write X509 cert
	err := lazyjack.CreateX509CertForCA(basePath)
	if err == nil {
		t.Errorf("FAILED: Expected that X509 cert could not be created")
	}
}

func TestBuildArgsForRSA(t *testing.T) {
	args := lazyjack.BuildArgsForRSA(lazyjack.WorkArea)
	actual := strings.Join(args, " ")
	expected := "rsa -pubin -in /tmp/lazyjack/certs/ca.x509 -outform der -out /tmp/lazyjack/certs/ca.rsa"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestFailingCreateRSAForCA(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)
	// No input file and no area to write RSA
	err := lazyjack.CreateRSAForCA(basePath)
	if err == nil {
		t.Errorf("FAILED: Expected that RSA could not be created")
	}
}

func TestBuildArgsForCADigest(t *testing.T) {
	args := lazyjack.BuildArgsForCADigest(lazyjack.WorkArea)
	actual := strings.Join(args, " ")
	expected := "dgst -sha256 -hex /tmp/lazyjack/certs/ca.rsa"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestExtractingDigest(t *testing.T) {
	var testCases = []struct {
		name        string
		input       string
		expected    string
		expectedErr string
	}{
		{
			name:        "Valid digest hash",
			input:       "SHA256(/tmp/lazyjack/certs/ca.rsa)= 134319a0d3333de4c2dd0f23d9a7647952e301ad81c56e2b016c6d636e445249",
			expected:    "134319a0d3333de4c2dd0f23d9a7647952e301ad81c56e2b016c6d636e445249",
			expectedErr: "",
		},
		{
			name:        "no hash in output",
			input:       "SHA256(/tmp/lazyjack/certs/ca.rsa)=",
			expected:    "",
			expectedErr: "Unable to parse digest info for CA key",
		},
		{
			name:        "hash invalid",
			input:       "SHA256(/tmp/lazyjack/certs/ca.rsa)= 134319a0d3333de4c2dd0f23d9a7647952e301ad81c56e2b016c6d636e44524",
			expected:    "",
			expectedErr: "Invalid token certificate hash length (63)",
		},
	}
	for _, tc := range testCases {
		actual, err := lazyjack.ExtractDigest(tc.input)
		if err != nil {
			if tc.expectedErr == "" {
				t.Errorf("FAILED: [%s] Error seen, when not expected: %s", tc.name, err.Error())
			} else if tc.expectedErr != err.Error() {
				t.Errorf("FAILED: [%s] Expected error %s, got %q", tc.name, tc.expectedErr, err.Error())
			}
		} else {
			if tc.expectedErr != "" {
				t.Errorf("FAILED: [%s] Expected error %s, but no error seen", tc.name, tc.expectedErr)
			} else if actual != tc.expected {
				t.Errorf("FAILED: [%s] Expected has %q, got %q", tc.name, tc.expected, actual)
			}
		}
	}
}

func TestFailingCreateDigestForCA(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)
	// No input file and no area to write digest
	_, err := lazyjack.CreateDigestForCA(basePath)
	if err == nil {
		t.Errorf("FAILED: Expected that CA digest could not be created")
	}
}

func TestCreatingAllCertsAndKeys(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	certArea := filepath.Join(basePath, lazyjack.CertArea)
	HelperSetupArea(certArea, t)
	defer HelperCleanupArea(certArea, t)

	err := lazyjack.CreateKeyForCA(basePath)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create CA key: %s", err.Error())
	}
	err = lazyjack.CreateCertificateForCA("fd00:100::", 2, basePath)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create CA cert: %s", err.Error())
	}
	_, err = lazyjack.CreateCertficateHashForCA(basePath)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create X509 cert, RSA, digest, and hash: %s", err.Error())
	}
}

func TestExtractToken(t *testing.T) {
	// NOTE: This test requires kubeadm to be installed on the system!
	var testCases = []struct {
		name        string
		input       string
		expected    string
		expectedErr string
	}{
		{
			name:        "Valid token",
			input:       "7aee33.05f81856d78346bd",
			expected:    "7aee33.05f81856d78346bd",
			expectedErr: "",
		},
		{
			name:        "Valid token with whitespace",
			input:       " 7aee33.05f81856d78346bd  ",
			expected:    "7aee33.05f81856d78346bd",
			expectedErr: "",
		},
		{
			name:        "token invalid",
			input:       "bad-token-value",
			expected:    "7aee33.05f81856d78346bd",
			expectedErr: "Internal error, token is malformed: Invalid token length (15)",
		},
	}
	for _, tc := range testCases {
		actual, err := lazyjack.ExtractToken(tc.input)
		if err != nil {
			if tc.expectedErr == "" {
				t.Errorf("FAILED: [%s] Error seen, when not expected: %s", tc.name, err.Error())
			} else if tc.expectedErr != err.Error() {
				t.Errorf("FAILED: [%s] Expected error %s, got %q", tc.name, tc.expectedErr, err.Error())
			}
		} else {
			if tc.expectedErr != "" {
				t.Errorf("FAILED: [%s] Expected error %s, but no error seen", tc.name, tc.expectedErr)
			} else if actual != tc.expected {
				t.Errorf("FAILED: [%s] Expected has %q, got %q", tc.name, tc.expected, actual)
			}
		}
	}
}

func HelperCreateConfigFile(filename string, t *testing.T) {
	contents := `#Sample for testing
general:
    plugin: bridge
topology:
    master:
        interface: "eth0"
        opmodes: "master dns64 nat64"
        id: 2
    minion-1:
        interface: "eth0"
        opmodes: "minion"
        id: 3
support_net:
    cidr: "fd00:10::/64"
    v4cidr: "172.18.0.0/16"
mgmt_net:
    cidr: "fd00:20::/64"
pod_net:
    prefix: "fd00:40:0:0"
    size: 80
service_net:
    cidr: "fd00:30::/110"
nat64:
    v4_cidr: "172.18.0.128/25"
    v4_ip: "172.18.0.200"
    ip: "fd00:10::200"
dns64:
    remote_server: "64.102.6.247"
    cidr: "fd00:10:64:ff9b::/96"
    ip: "fd00:10::100"
`
	err := ioutil.WriteFile(filename, []byte(contents), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create config file for test")
	}
}

func HelperReadConfig(configFile string, t *testing.T) *lazyjack.Config {
	cf, err := lazyjack.OpenConfigFile(configFile)
	if err != nil {
		t.Fatalf("ERROR: Unable to open config file for test")
	}
	config, err := lazyjack.LoadConfig(cf)
	if err != nil {
		t.Fatalf("ERROR: Unable to load config file for test")
	}
	err = lazyjack.ValidateConfigContents(config, true)
	if err != nil {
		t.Fatalf("ERROR: Unable to validate config file for test")
	}
	return config
}

func TestInitializeMaster(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	configFile := filepath.Join(basePath, "config.yaml")
	HelperCreateConfigFile(configFile, t)
	c := HelperReadConfig(configFile, t)
	// Override work area for files
	c.General.WorkArea = basePath

	err := lazyjack.Initialize("master", c, configFile)
	if err != nil {
		t.Errorf("FAILED: Unable to initialize: %s", err.Error())
	}
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		t.Errorf("FAILED: Unable to read config file, after processing: %s", err.Error())
	}
	if !bytes.Contains(contents, []byte("token:")) || !bytes.Contains(contents, []byte("token-cert-hash:")) {
		t.Errorf("FAILED: Expected config file to have token and cert")
	}
}

func TestInitializeMinion(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	configFile := filepath.Join(basePath, "config.yaml")
	HelperCreateConfigFile(configFile, t)
	c := HelperReadConfig(configFile, t)
	// Override work area for files
	c.General.WorkArea = basePath

	err := lazyjack.Initialize("minion-1", c, configFile)
	if err != nil {
		t.Errorf("FAILED: Unable to initialize: %s", err.Error())
	}
	// Should not modify config file on minion
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		t.Errorf("FAILED: Unable to read config file, after processing: %s", err.Error())
	}
	if bytes.Contains(contents, []byte("token:")) || bytes.Contains(contents, []byte("token-cert-hash:")) {
		t.Errorf("FAILED: Expected config file to not have been modified for minion")
	}
}

func TestFailingInitialize(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	configFile := filepath.Join(basePath, "config.yaml")
	HelperCreateConfigFile(configFile, t)
	c := HelperReadConfig(configFile, t)
	// Override work area for files
	c.General.WorkArea = basePath

	bogusConfigFile := filepath.Join(basePath, "no-such-config.yaml")
	err := lazyjack.Initialize("master", c, bogusConfigFile)
	if err == nil {
		t.Errorf("FAILED: Should not have been able to initialize, when config file is missing")
	}
}

func TestUpdateConfigYAMLContents(t *testing.T) {
	var testCases = []struct {
		name     string
		input    []byte
		token    string
		hash     string
		expected string
	}{
		{
			name: "not present - adding",
			input: bytes.NewBufferString(`# Adding new
general:
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Adding new
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "replacing existing",
			input: bytes.NewBufferString(`# Replacing
general:
    plugin: bridge
    token: "<provide-token-here>"
    token-cert-hash: "<provide-cert-hash-here>"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Replacing
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "replacing when duplicates (legacy)",
			input: bytes.NewBufferString(`# Duplicates
general:
    plugin: bridge
    token: "<provide-token-here>"
    token-cert-hash: "<provide-cert-hash-here>"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
token: "b362b2.665c96095a76fb5c"
token-cert-hash: "35f932d559ec963388046a690cdeaaced2408a16a2d3da529622c9dfb790fbe4"
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Duplicates
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "replacing when order diff",
			input: bytes.NewBufferString(`# Replacing diff order
general:
    plugin: bridge
    token-cert-hash: "<provide-cert-hash-here>"
    token: "<provide-token-here>"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Replacing diff order
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "replacing when first (legacy)",
			input: bytes.NewBufferString(`# Replacing first
token: "b362b2.665c96095a76fb5c"
token-cert-hash: "35f932d559ec963388046a690cdeaaced2408a16a2d3da529622c9dfb790fbe4"
general:
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Replacing first
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
    plugin: bridge
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "missing general line",
			input: bytes.NewBufferString(`# Adding new
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Adding new
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
`,
		},
		{
			name:  "empty file (invalid though)",
			input: bytes.NewBufferString("").Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `
general:
    token: "1a46e0.4623b882f4f887a2"
    token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
`,
		},
	}
	for _, tc := range testCases {
		actual := lazyjack.UpdateConfigYAMLContents(tc.input, "my-config.yaml", tc.token, tc.hash)
		if string(actual) != tc.expected {
			t.Errorf("FAILED: [%s] Incorrect contents.\nExpected:\n%s\nActual:\n%s\n", tc.name, tc.expected, actual)
		}
	}

}
