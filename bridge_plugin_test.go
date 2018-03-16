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
		t.Fatalf("FAILED: Bridge CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
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
		t.Fatalf("FAILED: Expected to be able to create CNI config file: %s", err.Error())
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
		t.Fatalf("FAILED: Expected to not be able to create CNI config file")
	}
	expected := "Unable to create CNI config for bridge plugin"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestDoRouteOpsOnNodesAdd(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "add")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to add route on node: %s", err.Error())
	}
}

func TestFailedDoRouteOpsOnNodesAdd(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "add")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "Unable to add pod network route for fd00:40:0:0:20::/80 to minion1: Mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedExistsDoRouteOpsOnNodesAdd(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteExists: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "add")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route - exists already")
	}
	expected := "Skipping - add route to fd00:40:0:0:20::/80 via fd00:100::20 as already exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestDoRouteOpsOnNodesDelete(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "delete")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete route on node: %s", err.Error())
	}
}

func TestFailedDoRouteOpsOnNodesDelete(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteDelFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "delete")
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete route on node")
	}
	expected := "Unable to delete pod network route for fd00:40:0:0:20::/80 to minion1: Mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNoRouteDoRouteOpsOnNodesDelete(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simNoRoute: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       20,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:      "minion1",
		Interface: "eth2",
		IsMinion:  true,
		ID:        20,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "delete")
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete route on node")
	}
	expected := "Skipping - delete route from fd00:40:0:0:10::/80 via fd00:100::10 as non-existent"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}
