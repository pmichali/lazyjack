package lazyjack_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestBuildArgsForCAKey(t *testing.T) {
	args := lazyjack.BuildArgsForCAKey()
	actual := strings.Join(args, " ")
	expected := "genrsa -out /tmp/lazyjack/certs/ca.key 2048"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestBuildArgsForCACert(t *testing.T) {
	n := &lazyjack.Node{ID: 2}
	c := &lazyjack.Config{
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}

	args := lazyjack.BuildArgsForCACert(n, c)
	actual := strings.Join(args, " ")
	expected := "req -x509 -new -nodes -key /tmp/lazyjack/certs/ca.key -subj /CN=fd00:100::2 -days 10000 -out /tmp/lazyjack/certs/ca.crt"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestBuildArgsForX509Cert(t *testing.T) {
	args := lazyjack.BuildArgsForX509Cert()
	actual := strings.Join(args, " ")
	expected := "x509 -pubkey -in /tmp/lazyjack/certs/ca.crt"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestBuildArgsForRSA(t *testing.T) {
	args := lazyjack.BuildArgsForRSA()
	actual := strings.Join(args, " ")
	expected := "rsa -pubin -in /tmp/lazyjack/certs/ca.x509 -outform der -out /tmp/lazyjack/certs/ca.rsa"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
	}
}

func TestBuildArgsForCADigest(t *testing.T) {
	args := lazyjack.BuildArgsForCADigest()
	actual := strings.Join(args, " ")
	expected := "dgst -sha256 -hex /tmp/lazyjack/certs/ca.rsa"
	if actual != expected {
		t.Errorf("FAILED: Arguments don't match. Expected %q, got %q", expected, actual)
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
plugin: bridge
token: "1a46e0.4623b882f4f887a2"
token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
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
plugin: bridge
token: "1a46e0.4623b882f4f887a2"
token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "replacing when duplicates",
			input: bytes.NewBufferString(`# Duplicates
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
plugin: bridge
token: "1a46e0.4623b882f4f887a2"
token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
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
plugin: bridge
token-cert-hash: "<provide-cert-hash-here>"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
token: "<provide-token-here>"
`).Bytes(),
			token: "1a46e0.4623b882f4f887a2",
			hash:  "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef",
			expected: `# Replacing diff order
plugin: bridge
token: "1a46e0.4623b882f4f887a2"
token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
`,
		},
		{
			name: "replacing when first",
			input: bytes.NewBufferString(`# Replacing first
token: "b362b2.665c96095a76fb5c"
token-cert-hash: "35f932d559ec963388046a690cdeaaced2408a16a2d3da529622c9dfb790fbe4"
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
plugin: bridge
token: "1a46e0.4623b882f4f887a2"
token-cert-hash: "05b24bf01253ff487504eeb264d4b018529e0430b9d9637cff374c39b740e7ef"
topology:
    bxb-c2-77:
        interface: "enp10s0"
        opmodes: "master dns64 nat64"
        id: 2
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
