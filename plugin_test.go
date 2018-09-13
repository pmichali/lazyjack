package lazyjack_test

import (
	"testing"

	"github.com/pmichali/lazyjack"
)

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

func TestBuildPodSubnetPrefix(t *testing.T) {
	var testCases = []struct {
		name           string
		prefix         string
		size           int
		node_id        int
		mode           string
		expectedPrefix string
		expectedSuffix string
	}{
		{
			name:           "node in lower byte, no upper byte",
			prefix:         "fd00:40:0:0:",
			size:           80,
			node_id:        10,
			mode:           "ipv6",
			expectedPrefix: "fd00:40:0:0:a::",
			expectedSuffix: "",
		},
		{
			name:           "node in upper byte",
			prefix:         "fd00:40:0:0:",
			size:           72,
			node_id:        10,
			mode:           "ipv6",
			expectedPrefix: "fd00:40:0:0:a00::",
			expectedSuffix: "",
		},
		{
			name:           "node added to lower byte",
			prefix:         "fd00:10:20:30:40",
			size:           80,
			node_id:        02,
			mode:           "ipv6",
			expectedPrefix: "fd00:10:20:30:4002::",
			expectedSuffix: "",
		},
		{
			name:           "ipv4 /24 only",
			prefix:         "10.244.0.",
			size:           24,
			node_id:        20,
			mode:           "ipv4",
			expectedPrefix: "10.244.20.",
			expectedSuffix: "0",
		},
	}
	for _, tc := range testCases {
		actualPrefix, actualSuffix := lazyjack.BuildPodSubnetPrefix(tc.mode, tc.prefix, tc.size, tc.node_id)
		if actualPrefix != tc.expectedPrefix {
			t.Errorf("[%s] Expected prefix: %q, got %q", tc.name, tc.expectedPrefix, actualPrefix)
		}
		if actualSuffix != tc.expectedSuffix {
			t.Errorf("[%s] Expected prefix: %q, got %q", tc.name, tc.expectedSuffix, actualSuffix)
		}
	}
}
