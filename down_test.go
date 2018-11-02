package lazyjack_test

import (
	"fmt"
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

	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
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
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:0:0",
					Size:   80,
				},
			},
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
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
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

	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
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
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:100::",
				},
			},
		},
		Pod: lazyjack.PodNetwork{
			Info: [2]lazyjack.NetInfo{
				{
					Prefix: "fd00:40:0:0",
					Size:   80,
				},
			},
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

// HelperKubeAdmResetExecCommand mosks OS command requests for kubeadm reset
func HelperKubeAdmResetExecCommand(cmd string, args []string) (string, error) {
	if cmd == "kubeadm" {
		if len(args) == 2 && args[0] == "reset" && args[1] == "-f" {
			return "Mocked kubeadm reset worked", nil
		} else {
			return "", fmt.Errorf("Wrong args for kubeadm command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking kubeadm command only")
}

func TestStopKubernetes(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperKubeAdmResetExecCommand)
	err := lazyjack.StopKubernetes()
	if err != nil {
		t.Fatalf("Should have been able to stop kubernetes cluster (mocked): %s", err.Error())
	}
}

// HelperKubeAdmResetFailExecCommand mosks OS command requests for kubeadm reset
func HelperKubeAdmResetFailExecCommand(cmd string, args []string) (string, error) {
	if cmd == "kubeadm" {
		if len(args) == 2 && args[0] == "reset" && args[1] == "-f" {
			return "", fmt.Errorf("mock failure")
		} else {
			return "", fmt.Errorf("Wrong args for kubeadm command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking kubeadm command only")
}

func TestFailStopKubernetes(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperKubeAdmResetFailExecCommand)
	err := lazyjack.StopKubernetes()
	if err == nil {
		t.Fatalf("Expected to fail kubeadm reset command, but was successful")
	}
	expected := "unable to reset Kubernetes cluster: mock failure"
	if err.Error() != expected {
		t.Fatalf("Expected failure to be %q, but got %q", expected, err.Error())
	}
}
