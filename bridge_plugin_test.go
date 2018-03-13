package lazyjack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestBridgeCNIConfigContents(t *testing.T) {
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
	}
	n := &lazyjack.Node{ID: 10}

	expected := `{
    "cniVersion": "0.3.0",
    "name": "bmbridge",
    "type": "bridge",
    "bridge": "br0",
    "isDefaultGateway": true,
    "ipMasq": true,
    "hairpinMode": true,
    "ipam": {
        "type": "host-local",
        "ranges": [
          [
            {
              "subnet": "fd00:40:0:0:10::/80",
              "gateway": "fd00:40:0:0:10::1"
	    }
          ]
        ]
    }
}
`
	actual := lazyjack.CreateBridgeCNIConfContents(n, c)
	if actual.String() != expected {
		t.Errorf("FAILED: Bridge CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
	}
}

func TestCreateBridgeCNIConfigFile(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			CNIArea: cniArea,
		},
	}
	n := &lazyjack.Node{ID: 10}

	err := lazyjack.CreateBridgeCNIConfigFile(n, c)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to create CNI config file: %s", err.Error())
	}
}

func TestFailedCreateBridgeCNIConfigFile(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

	// Create a non-writeable CNI config file to cause failure when writing
	filename := filepath.Join(cniArea, lazyjack.CNIConfFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0400)
	if err != nil {
		t.Fatalf("ERROR: Unable to create CNI config file for test")
	}

	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			CNIArea: cniArea,
		},
	}
	n := &lazyjack.Node{ID: 10}

	err = lazyjack.CreateBridgeCNIConfigFile(n, c)
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to create CNI config file")
	}
	expected := "Unable to create CNI config for bridge plugin"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}
