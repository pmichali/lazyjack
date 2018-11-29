package lazyjack_test

import (
	"bytes"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestPointToPointPluginCleanup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
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
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:0:0:",
					Size:   80,
				},
			},
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	// Currently, we expect NAT64 node to also be DNS64 node.
	n := &lazyjack.Node{
		Name:      "master",
		ID:        0x10,
		Interface: "eth1",
		IsMaster:  true,
	}

	err := c.General.CNIPlugin.Cleanup(n)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to cleanup PTP plugin")
	}
}

func TestFailedPointToPointPluginCleanup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteDelFail: true}}
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
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:0:0:",
					Size:   80,
				},
			},
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	// Currently, we expect NAT64 node to also be DNS64 node.
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
	expected := "unable to remove routes for PTP plugin: unable to delete pod network route for fd00:40:0:0:20::/80 to minion1: mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestPointToPointPluginSetup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       0x10,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       0x60,
			},
		},
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:30:20:",
					Size:   72,
				},
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        0x60,
	}

	err := c.General.CNIPlugin.Setup(n)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to setup PTP plugin")
	}
}

func TestFailedPointToPointPluginSetup(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteAddFail: true}}
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMinion: true,
				Name:     "minion1",
				ID:       0x10,
			},
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       0x60,
			},
		},
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:30:20:",
					Size:   72,
				},
			},
		},
		General: lazyjack.GeneralSettings{
			NetMgr: nm,
		},
		Mgmt: lazyjack.ManagementNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	n := &lazyjack.Node{
		Name:      "master",
		Interface: "eth1",
		IsMaster:  true,
		ID:        0x60,
	}

	err := c.General.CNIPlugin.Setup(n)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "unable to add pod network route for fd00:40:30:20:1000::/72 to minion1: mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestPointToPointCNIConfigContents(t *testing.T) {
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:0:0:",
					Mode:   lazyjack.IPv6NetMode,
					Size:   80,
				},
			},
			MTU: 9000,
		},
		General: lazyjack.GeneralSettings{
			Mode: "ipv6",
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	n := &lazyjack.Node{ID: 10}

	expected := `{
  "cniVersion": "0.3.1",
  "name": "dindnet",
  "type": "ptp",
  "ipMasq": true,
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
    ],
    "routes": [
      {"dst": "::/0"}
    ]
  }
}
`
	actual := new(bytes.Buffer)
	err := c.General.CNIPlugin.WriteConfigContents(n, actual)
	if err != nil {
		t.Fatalf("FAILED! Expected to be able to write CNI configuration %s", err.Error())
	}
	if actual.String() != expected {
		t.Fatalf("FAILED: PTP CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
	}
}

func TestPointToPointCNIConfigContentsV4(t *testing.T) {
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "10.244.0.",
					Mode:   lazyjack.IPv4NetMode,
					Size:   24,
				},
			},
			MTU: 1500,
		},
		General: lazyjack.GeneralSettings{
			Mode: "ipv4",
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	n := &lazyjack.Node{ID: 10}

	expected := `{
  "cniVersion": "0.3.1",
  "name": "dindnet",
  "type": "ptp",
  "ipMasq": true,
  "mtu": 1500,
  "ipam": {
    "type": "host-local",
    "ranges": [
      [
        {
          "subnet": "10.244.10.0/24",
          "gateway": "10.244.10.1"
        }
      ]
    ],
    "routes": [
      {"dst": "0.0.0.0/0"}
    ]
  }
}
`
	actual := new(bytes.Buffer)
	err := c.General.CNIPlugin.WriteConfigContents(n, actual)
	if err != nil {
		t.Fatalf("FAILED! Expected to be able to write CNI configuration %s", err.Error())
	}
	if actual.String() != expected {
		t.Fatalf("FAILED: PTP CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
	}
}

func TestPointToPointCNIConfigContentsDualStack(t *testing.T) {
	c := &lazyjack.Config{
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:0:0:",
					Mode:   lazyjack.IPv6NetMode,
					Size:   80,
				},
				{
					Prefix: "10.244.0.",
					Mode:   lazyjack.IPv4NetMode,
					Size:   24,
				},
			},
			MTU: 9000,
		},
		General: lazyjack.GeneralSettings{
			Mode: "dual-stack",
		},
	}
	c.General.CNIPlugin = lazyjack.PointToPointPlugin{c}
	n := &lazyjack.Node{ID: 10}

	expected := `{
  "cniVersion": "0.3.1",
  "name": "dindnet",
  "type": "ptp",
  "ipMasq": true,
  "mtu": 9000,
  "ipam": {
    "type": "host-local",
    "ranges": [
      [
        {
          "subnet": "fd00:40:0:0:a::/80",
          "gateway": "fd00:40:0:0:a::1"
        }
      ],
      [
        {
          "subnet": "10.244.10.0/24",
          "gateway": "10.244.10.1"
        }
      ]
    ],
    "routes": [
      {"dst": "::/0"},
      {"dst": "0.0.0.0/0"}
    ]
  }
}
`
	actual := new(bytes.Buffer)
	err := c.General.CNIPlugin.WriteConfigContents(n, actual)
	if err != nil {
		t.Fatalf("FAILED! Expected to be able to write CNI configuration %s", err.Error())
	}
	if actual.String() != expected {
		t.Fatalf("FAILED: PTP CNI config contents wrong\nExpected:\n%s\n  Actual:\n%s\n", expected, actual.String())
	}
}
