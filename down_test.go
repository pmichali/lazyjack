package lazyjack_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestCleanupForPlugin(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
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

	lazyjack.CleanupForPlugin(n, c)
}
