package lazyjack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestBuildPodSubnetPrefix(t *testing.T) {
	var testCases = []struct {
		name     string
		prefix   string
		size     int
		node_id  int
		expected string
	}{
		{
			name:     "node in lower byte, no upper byte",
			prefix:   "fd00:40:0:0:",
			size:     80,
			node_id:  10,
			expected: "fd00:40:0:0:a::",
		},
		{
			name:     "node in upper byte",
			prefix:   "fd00:40:0:0:",
			size:     72,
			node_id:  10,
			expected: "fd00:40:0:0:a00::",
		},
		{
			name:     "node added to lower byte",
			prefix:   "fd00:10:20:30:40",
			size:     80,
			node_id:  02,
			expected: "fd00:10:20:30:4002::",
		},
	}
	for _, tc := range testCases {
		actual := lazyjack.BuildPodSubnetPrefix(tc.prefix, tc.size, tc.node_id)
		if actual != tc.expected {
			t.Errorf("[%s] Expected: %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestBridgeCNIConfigContents(t *testing.T) {
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Prefix: "fd00:40:0:0:",
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
              "subnet": "fd00:40:0:0:a::/80",
              "gateway": "fd00:40:0:0:a::1"
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
			Prefix: "fd00:40:0:0:",
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
			Prefix: "fd00:40:0:0:",
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
	expected := "unable to create CNI config for bridge plugin"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestDoRouteOpsOnNodesAdd(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
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
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        0x10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "add")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "unable to add pod network route for fd00:40:0:0:20::/80 to minion1: mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedExistsDoRouteOpsOnNodesAdd(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteExists: true}}
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
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        0x10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "add")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route - exists already")
	}
	expected := "skipping - add route to fd00:40:0:0:20::/80 via fd00:100::20 as already exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestDoRouteOpsOnNodesDelete(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{}}
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
	nm := lazyjack.NetMgr{Server: mockNetLink{simRouteDelFail: true}}
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
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        0x10,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "delete")
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete route on node")
	}
	expected := "unable to delete pod network route for fd00:40:0:0:20::/80 to minion1: mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNoRouteDoRouteOpsOnNodesDelete(t *testing.T) {
	nm := lazyjack.NetMgr{Server: mockNetLink{simNoRoute: true}}
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
	n := &lazyjack.Node{
		Name:      "minion1",
		Interface: "eth2",
		IsMinion:  true,
		ID:        0x20,
	}

	err := lazyjack.DoRouteOpsOnNodes(n, c, "delete")
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to delete route on node")
	}
	expected := "skipping - delete route from fd00:40:0:0:10::/80 via fd00:100::10 as non-existent"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}
