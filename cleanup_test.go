package lazyjack_test

import (
	"bytes"
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
