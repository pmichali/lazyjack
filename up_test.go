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

func SlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildKubeAdmCommand(t *testing.T) {
	c := &lazyjack.Config{
		General: lazyjack.GeneralSettings{
			Token:         "<valid-token-here>",
			TokenCertHash: "<valid-ca-certificate-hash-here>",
			WorkArea:      "/some/work/area",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	minionNode := &lazyjack.Node{
		ID:       20,
		IsMaster: false,
	}
	masterNode := &lazyjack.Node{
		ID:       10,
		IsMaster: true,
	}
	actual := lazyjack.BuildKubeAdmCommand(masterNode, masterNode, c)
	expected := []string{"init", "--config=/some/work/area/kubeadm.conf"}

	if !SlicesEqual(actual, expected) {
		t.Errorf("KubeAdm init args incorrect for master node. Expected %q, got %q", strings.Join(expected, " "), strings.Join(actual, " "))
	}

	actual = lazyjack.BuildKubeAdmCommand(minionNode, masterNode, c)
	expected = []string{"join", "--token", "<valid-token-here>",
		"--discovery-token-ca-cert-hash", "sha256:<valid-ca-certificate-hash-here>",
		"[fd00:100::10]:6443"}

	if !SlicesEqual(actual, expected) {
		t.Errorf("KubeAdm init args incorrect for minion node. Expected %q, got %q", strings.Join(expected, " "), strings.Join(actual, " "))
	}

	c.General.Insecure = true
	actual = lazyjack.BuildKubeAdmCommand(minionNode, masterNode, c)
	expected = []string{"join", "--token", lazyjack.DefaultToken,
		"--discovery-token-unsafe-skip-ca-verification=true",
		"--ignore-preflight-errors=all", "[fd00:100::10]:6443"}

	if !SlicesEqual(actual, expected) {
		t.Errorf("KubeAdm init args incorrect for insecure minion node. Expected %q, got %q", strings.Join(expected, " "), strings.Join(actual, " "))
	}
}

func TestEnsureCNIAreaExists(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	err := lazyjack.EnsureCNIAreaExists(basePath)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create CNI area in %q: %s", basePath, err.Error())
	}
}

func TestFailingEnsureCNIAreaExists(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Set CNI base a level lower, so that we can make parent read-only, preventing deletion
	cniBase := filepath.Join(basePath, "dummy")
	err := os.MkdirAll(cniBase, 0700)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	err = os.Chmod(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	defer func() { os.Chmod(basePath, 0700) }()

	err = lazyjack.EnsureCNIAreaExists(cniBase)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to create CNI area in %q", cniBase)
	}
}

func TestCopyFile(t *testing.T) {
	srcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(srcArea, t)
	defer HelperCleanupArea(srcArea, t)

	dstArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(dstArea, t)
	defer HelperCleanupArea(dstArea, t)

	// Create a valid source file
	filename := filepath.Join(srcArea, "foo")
	err := ioutil.WriteFile(filename, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	err = lazyjack.CopyFile("foo", srcArea, dstArea)
	if err != nil {
		t.Fatalf("FAILURE: Expected to be able to copy file: %s", err.Error())
	}
}

func TestCopyFileFailures(t *testing.T) {
	srcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(srcArea, t)
	defer HelperCleanupArea(srcArea, t)

	dstArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(dstArea, t)
	defer HelperCleanupArea(dstArea, t)

	// No source file...
	err := lazyjack.CopyFile("foo", srcArea, dstArea)
	if err == nil {
		t.Fatalf("FAILURE: Expected not to be able to copy non-existing source file")
	}
	expected := "unable to open source file \"foo\":"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILURE: Expected error message %q, got %q", expected, err.Error())
	}

	// Create a valid source file
	filename := filepath.Join(srcArea, "foo")
	err = ioutil.WriteFile(filename, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	// Create a file that is read-only, so that it cannot be overwritten
	filename = filepath.Join(dstArea, "foo")
	err = ioutil.WriteFile(filename, []byte("# empty file"), 0400)
	if err != nil {
		t.Fatalf("ERROR: Unable to create dest file for test")
	}

	// Unable to copy to dest...
	err = lazyjack.CopyFile("foo", srcArea, dstArea)
	if err == nil {
		t.Fatalf("FAILURE: Expected not to be able to copy non-existing source file")
	}
	expected = "unable to open destination file \"foo\":"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILURE: Expected error message %q, got %q", expected, err.Error())
	}
}

func TestCopyFileNoSource(t *testing.T) {
	srcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(srcArea, t)
	defer HelperCleanupArea(srcArea, t)

	dstArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(dstArea, t)
	defer HelperCleanupArea(dstArea, t)

	err := lazyjack.CopyFile("foo", srcArea, dstArea)
	if err == nil {
		t.Fatalf("FAILURE: Expected not to be able to copy non-existing source file")
	}
	expected := "unable to open source file \"foo\":"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILURE: Expected error message %q, got %q", expected, err.Error())
	}
}

func TestCopyFileDestReadOnly(t *testing.T) {
	srcArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(srcArea, t)
	defer HelperCleanupArea(srcArea, t)

	dstArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(dstArea, t)
	defer HelperCleanupArea(dstArea, t)

	// Create a valid source file
	filename := filepath.Join(srcArea, "foo")
	err := ioutil.WriteFile(filename, []byte("# dummy file"), 0700)
	if err != nil {
		t.Fatalf("ERROR: Unable to create source file for test")
	}

	// Create a file that is read-only, so that it cannot be overwritten
	filename = filepath.Join(dstArea, "foo")
	err = ioutil.WriteFile(filename, []byte("# empty file"), 0400)
	if err != nil {
		t.Fatalf("ERROR: Unable to create dest file for test")
	}

	// Unable to copy to dest...
	err = lazyjack.CopyFile("foo", srcArea, dstArea)
	if err == nil {
		t.Fatalf("FAILURE: Expected not to be able to copy non-existing source file")
	}
	expected := "unable to open destination file \"foo\":"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILURE: Expected error message %q, got %q", expected, err.Error())
	}
}

func TestPlaceCertificateAndKeyForCA(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	srcPath := filepath.Join(basePath, lazyjack.CertArea)
	HelperSetupArea(srcPath, t)
	defer HelperCleanupArea(basePath, t)

	cert := filepath.Join(basePath, lazyjack.CertArea, "ca.crt")
	err := ioutil.WriteFile(cert, []byte("# dummy file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create ca.crt file for test")
	}
	key := filepath.Join(basePath, lazyjack.CertArea, "ca.key")
	err = ioutil.WriteFile(key, []byte("# dummy file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create ca.key file for test")
	}

	dstPath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(dstPath, t)

	err = lazyjack.PlaceCertificateAndKeyForCA(basePath, dstPath)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to copy certs from %q to %q: %s", basePath, dstPath, err.Error())
	}
	certCopy := filepath.Join(dstPath, "ca.crt")
	if _, err := os.Stat(certCopy); os.IsNotExist(err) {
		t.Fatalf("FAILED: ca.crt was not copied")
	}
	keyCopy := filepath.Join(dstPath, "ca.key")
	if _, err := os.Stat(keyCopy); os.IsNotExist(err) {
		t.Fatalf("FAILED: ca.key was not copied")
	}
}

func TestFailingPlaceCertificateAndKeyForCANotWriteabe(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	srcPath := filepath.Join(basePath, lazyjack.CertArea)
	HelperSetupArea(srcPath, t)
	defer HelperCleanupArea(basePath, t)

	cert := filepath.Join(basePath, lazyjack.CertArea, "ca.crt")
	err := ioutil.WriteFile(cert, []byte("# dummy file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create ca.crt file for test")
	}
	key := filepath.Join(basePath, lazyjack.CertArea, "ca.key")
	err = ioutil.WriteFile(key, []byte("# dummy file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create ca.key file for test")
	}

	dstArea := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(dstArea, t)

	// Set destination a level lower, so that we can make parent read-only, preventing deletion
	dstPath := filepath.Join(dstArea, "dummy")
	err = os.MkdirAll(dstArea, 0700)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	err = os.Chmod(dstArea, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	defer func() { os.Chmod(dstArea, 0700) }()

	err = lazyjack.PlaceCertificateAndKeyForCA(basePath, dstPath)
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to copy certs from %q to %q", basePath, dstPath)
	}
}

func TestFailingPlaceCertificateAndKeyForCAMissingKeyFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	srcPath := filepath.Join(basePath, lazyjack.CertArea)
	HelperSetupArea(srcPath, t)
	defer HelperCleanupArea(basePath, t)

	key := filepath.Join(basePath, lazyjack.CertArea, "ca.crt")
	err := ioutil.WriteFile(key, []byte("# dummy file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create ca.crt file for test")
	}

	dstPath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(dstPath, t)

	err = lazyjack.PlaceCertificateAndKeyForCA(basePath, dstPath)
	if err == nil {
		t.Fatalf("FAILED: Expected missing ca.key file for copy")
	}
}

func TestFailingPlaceCertificateAndKeyForCAMissingCrtFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	srcPath := filepath.Join(basePath, lazyjack.CertArea)
	HelperSetupArea(srcPath, t)
	defer HelperCleanupArea(basePath, t)

	key := filepath.Join(basePath, lazyjack.CertArea, "ca.key")
	err := ioutil.WriteFile(key, []byte("# dummy file"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to create ca.key file for test")
	}

	dstPath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(dstPath, t)

	err = lazyjack.PlaceCertificateAndKeyForCA(basePath, dstPath)
	if err == nil {
		t.Fatalf("FAILED: Expected missing ca.crt file for copy")
	}
}

func TestDetermineMasterNode(t *testing.T) {
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMaster: false,
			},
			"master": {
				IsMaster: true,
			},
			"minion2": {
				IsMaster: false,
			},
		},
	}
	n := lazyjack.DetermineMasterNode(c)
	if n == nil {
		t.Fatalf("FAILED: Expected there to be a master node")
	}

	c = &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion1": {
				IsMaster: false,
			},
			"minion2": {
				IsMaster: false,
			},
		},
	}
	n = lazyjack.DetermineMasterNode(c)
	if n != nil {
		t.Fatalf("FAILED: Expected there NOT to be a master node")
	}

}

func TestSetupForPlugin(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

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
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
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
		ID:        10,
	}

	err := lazyjack.SetupForPlugin(n, c)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create config file and routes: %s", err.Error())
	}
}

func TestFailedNoCNIAreaSetupForPlugin(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	defer HelperCleanupArea(basePath, t)

	// Set CNI base a level lower, so that we can make parent read-only, preventing deletion
	cniBase := filepath.Join(basePath, "dummy")
	err := os.MkdirAll(cniBase, 0700)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	err = os.Chmod(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure: %s", err.Error())
	}
	defer func() { os.Chmod(basePath, 0700) }()

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
			Prefix: "fd00:40:0:0",
			Size:   80,
		},
		General: lazyjack.GeneralSettings{
			NetMgr:  nm,
			CNIArea: cniBase,
			Plugin:  "bridge",
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
		ID:        10,
	}

	err = lazyjack.SetupForPlugin(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create CNI area")
	}
	expected := "permission denied"
	if !strings.HasSuffix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedRouteCreateSetupForPlugin(t *testing.T) {
	cniArea := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(cniArea, t)
	defer HelperCleanupArea(cniArea, t)

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
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
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

	err := lazyjack.SetupForPlugin(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "unable to add pod network route for fd00:40:0:0:20::/80 to minion1: mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

// HelperSystemctlExecCommand mosks OS command requests for systemctl
func HelperSystemctlExecCommand(cmd string, args []string) (string, error) {
	if cmd == "systemctl" {
		if len(args) == 2 && args[0] == "restart" && args[1] == "kubelet" {
			return "Mocked systemctl restart kubelet worked", nil
		} else if len(args) == 1 && args[0] == "daemon-reload" {
			return "Mocked systemctl daemon-reload worked", nil
		} else {
			return "", fmt.Errorf("Wrong args for systemctl command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking systemctl command only")
}

func TestRestartKubeletService(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperSystemctlExecCommand)
	err := lazyjack.RestartKubeletService()
	if err != nil {
		t.Fatalf("Should have been able to restart kubelet service (mocked): %s", err.Error())
	}
}

// HelperSystemctlRestartFailureExecCommand mosks OS command requests for systemctl
func HelperSystemctlRestartFailureExecCommand(cmd string, args []string) (string, error) {
	if cmd == "systemctl" {
		if len(args) == 2 && args[0] == "restart" && args[1] == "kubelet" {
			return "", fmt.Errorf("mock failure")
		} else if len(args) == 1 && args[0] == "daemon-reload" {
			return "Mocked systemctl daemon-reload worked", nil
		} else {
			return "", fmt.Errorf("Wrong args for systemctl command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking systemctl command only")
}

func TestFailureRestartKubeletService(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperSystemctlRestartFailureExecCommand)
	err := lazyjack.RestartKubeletService()
	if err == nil {
		t.Fatalf("Expected to fail systemctl kubelet restart command, but was successful")
	}
	expected := "unable to restart kubelet service: mock failure"
	if err.Error() != expected {
		t.Fatalf("Expected failure to be %q, but got %q", expected, err.Error())
	}
}

// HelperSystemctlReloadFailureExecCommand mosks OS command requests for systemctl
func HelperSystemctlReloadFailureExecCommand(cmd string, args []string) (string, error) {
	if cmd == "systemctl" {
		if len(args) == 2 && args[0] == "restart" && args[1] == "kubelet" {
			return "Mocked systemctl restart kubelet worked", nil
		} else if len(args) == 1 && args[0] == "daemon-reload" {
			return "", fmt.Errorf("mock failure")
		} else {
			return "", fmt.Errorf("Wrong args for systemctl command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking systemctl command only")
}

func TestFailureReloadKubeletService(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperSystemctlReloadFailureExecCommand)
	err := lazyjack.RestartKubeletService()
	if err == nil {
		t.Fatalf("Expected to fail systemctl daemon reload command, but was successful")
	}
	expected := "unable to reload daemons: mock failure"
	if err.Error() != expected {
		t.Fatalf("Expected failure to be %q, but got %q", expected, err.Error())
	}
}

// HelperKubeAdmInitExecCommand will mock the OS command requests for kubeadm init
func HelperKubeAdmInitExecCommand(cmd string, args []string) (string, error) {
	if cmd == "kubeadm" {
		if len(args) == 2 && args[0] == "init" && args[1] == "--config=/tmp/kubeadm.conf" {
			return "Mocked kubeadm init worked", nil
		} else {
			return "", fmt.Errorf("Wrong args for kubeadm command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking kubeadm command only")
}

func TestStartKubernetes(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperKubeAdmInitExecCommand)
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		General: lazyjack.GeneralSettings{
			WorkArea:      "/tmp",
			Token:         "dummy-token",
			TokenCertHash: "dummy-hash-here",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:     "master",
		IsMaster: true,
		ID:       10,
	}

	err := lazyjack.StartKubernetes(n, c)
	if err != nil {
		t.Fatalf("Should have been able to start kubernetes (mocked): %s", err.Error())
	}
}

func TestFailedStartKubernetesNoMaster(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperKubeAdmInitExecCommand)
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"minion": {
				IsMaster: false,
				Name:     "master",
				ID:       10,
			},
		},
		General: lazyjack.GeneralSettings{
			WorkArea:      "/tmp",
			Token:         "dummy-token",
			TokenCertHash: "dummy-hash-here",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:     "minion",
		IsMaster: false,
		ID:       10,
	}

	err := lazyjack.StartKubernetes(n, c)
	if err == nil {
		t.Fatalf("Expected to fail, because no master node, but was successful")
	}
	expected := "unable to determine master node"
	if err.Error() != expected {
		t.Fatalf("Expected failure to be %q, but got %q", expected, err.Error())
	}
}

// HelperKubeAdmInitFailExecCommand will mock the OS command requests for kubeadm init failure
func HelperKubeAdmInitFailExecCommand(cmd string, args []string) (string, error) {
	if cmd == "kubeadm" {
		if len(args) == 2 && args[0] == "init" && args[1] == "--config=/tmp/kubeadm.conf" {
			return "", fmt.Errorf("unable to init Kubernetes cluster: failed running \"kubeadm\" with args \"init --config=/home/c2/bare-metal/work-area/kubeadm.conf\": exit status 2")
		} else {
			return "", fmt.Errorf("Wrong args for kubeadm command: %v", args)
		}
	}
	return "", fmt.Errorf("Test setup error - expected to be mocking kubeadm command only")
}

func TestFailedStartKubernetes(t *testing.T) {
	lazyjack.RegisterExecCommand(HelperKubeAdmInitFailExecCommand)
	c := &lazyjack.Config{
		Topology: map[string]lazyjack.Node{
			"master": {
				IsMaster: true,
				Name:     "master",
				ID:       10,
			},
		},
		General: lazyjack.GeneralSettings{
			WorkArea:      "/tmp",
			Token:         "dummy-token",
			TokenCertHash: "dummy-hash-here",
		},
		Mgmt: lazyjack.ManagementNetwork{
			Prefix: "fd00:100::",
		},
	}
	n := &lazyjack.Node{
		Name:     "master",
		IsMaster: true,
		ID:       10,
	}

	err := lazyjack.StartKubernetes(n, c)
	if err == nil {
		t.Fatalf("Expected to fail init command, but was successful")
	}
	expected := "unable to init Kubernetes cluster: unable to init Kubernetes cluster: failed running \"kubeadm\" with args \"init --config=/home/c2/bare-metal/work-area/kubeadm.conf\": exit status 2"
	if err.Error() != expected {
		t.Fatalf("Expected failure to be %q, but got %q", expected, err.Error())
	}
}
