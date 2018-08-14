package lazyjack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestCleanupForPlugin(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				Name:     "master",
				ID:       10,
				IsMaster: true,
			},
			"minion1": {
				Name:     "minion1",
				ID:       20,
				IsMinion: true,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:      "master",
		ID:        10,
		Interface: "eth1",
		IsMaster:  true,
	}

	filename := filepath.Join(cniArea, lazyjack.CNIConfFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create dummy CNI config file for test")
	}

	err = lazyjack.CleanupForPlugin(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to clean up for plugin: %s", err.Error())
	}
}

func TestFailedRemovingRouteCleanupForPlugin(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

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
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
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
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:      "master",
		ID:        0x10,
		Interface: "eth1",
		IsMaster:  true,
	}

	filename := filepath.Join(cniArea, lazyjack.CNIConfFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create dummy CNI config file for test")
	}

	err = lazyjack.CleanupForPlugin(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to remove route")
	}
	expected := "unable to remove routes for bridge plugin: unable to delete pod network route for fd00:40:0:0:20::/80 to minion1: mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestFailedRemoveFileCleanupForPlugin(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

	nm := lazyjack.NetMgr{Server: mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				Name:     "master",
				ID:       10,
				IsMaster: true,
			},
			"minion1": {
				Name:     "minion1",
				ID:       20,
				IsMinion: true,
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
	}
	c.General.CNIPlugin = lazyjack.BridgePlugin{c}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:      "master",
		ID:        10,
		Interface: "eth1",
		IsMaster:  true,
	}

	filename := filepath.Join(cniArea, lazyjack.CNIConfFile)
	err := ioutil.WriteFile(filename, []byte("# empty file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create dummy CNI config file for test")
	}
	// Cause it to fail, when removing area
	HelperMakeReadOnly(cniArea, t)
	defer HelperMakeWriteable(cniArea, t)

	err = lazyjack.CleanupForPlugin(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to remove CNI config area")
	}
	expected := "unable to remove CNI config file and area"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}
