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
			Prefix: "fd00:40:0:0:",
			Size:   80,
			MTU:    9000,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	n := &lazyjack.Node{ID: 10}

	expected := `{
    "cniVersion": "0.3.1",
    "name": "bmbridge",
    "type": "bridge",
    "bridge": "br0",
    "isDefaultGateway": true,
    "ipMasq": true,
    "hairpinMode": true,
    "mtu": 9000,
    "ipam": {
        "type": "host-local",
        "ranges": [
          [
            {
              "subnet": "fd00:40:0:0:a::/80",
              "gateway": "fd00:40:0:0:a::1"
	    }
          ]
        ]
    }
}
`
	actual := c.General.CNIPlugin.ConfigContents(n)
	if actual.String() != expected {
		t.Fatalf("FAILED: Bridge CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
	}
}

func TestCreateBridgeCNIConfigFile(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0:",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			Plugin:  "bridge",
			CNIArea: cniArea,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	n := &lazyjack.Node{ID: 10}

	err := lazyjack.CreateCNIConfigFile(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create CNI config file: %s", err.Error())
	}
}

func TestFailedSetupUnableToCreateBridgeCNIConfigFile(t *testing.T) {
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
			Prefix: "fd00:40:0:0:",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			Plugin:  "bridge",
			CNIArea: cniArea,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	n := &lazyjack.Node{ID: 10}

	err = lazyjack.CreateCNIConfigFile(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create CNI config file")
	}
	expected := "unable to create CNI config for bridge plugin"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestFailedBridgePluginSetup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       0x20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       0x10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0:",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        0x10,
	}

	err := c.General.CNIPlugin.Setup(n)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "unable to add pod network route for fd00:40:0:0:20::/80 to minion1: mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRemoveRoutesBridgePluginCleanup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteDelFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				Name:     "master",
				ID:       0x10,
				IsMaster: true,
			},
			"minion1": {
				Name:     "minion1",
				ID:       0x20,
				IsMinion: true,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
			Plugin: "bridge",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0:",
			Size:   80,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	n := &lazyjack.Node{
		Name:      "master",
		ID:        0x10,
		Interface: "eth1",
		IsMaster:  true,
	}

	err := c.General.CNIPlugin.Cleanup(n)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to remove route")
	}
	expected := "unable to remove routes for bridge plugin: unable to delete pod network route for fd00:40:0:0:20::/80 to minion1: mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestFailedBridgeRemoveBridgePluginCleanup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simSetDownFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				Name:     "master",
				ID:       0x10,
				IsMaster: true,
			},
			"minion1": {
				Name:     "minion1",
				ID:       0x20,
				IsMinion: true,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
			Plugin: "bridge",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0:",
			Size:   80,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	n := &lazyjack.Node{
		Name:      "master",
		ID:        0x10,
		Interface: "eth1",
		IsMaster:  true,
	}

	err := c.General.CNIPlugin.Cleanup(n)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to remove bridge")
	}
	expected := "unable to remove br0 bridge: unable to shut down interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}
