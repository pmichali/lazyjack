package lazyjack_test

import (
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
		//	"[fd00:100::10]:6443", "--discovery-token-unsafe-skip-ca-verification"}
		"[fd00:100::10]:6443", "--discovery-token-ca-cert-hash", "sha256:<valid-ca-certificate-hash-here>"}

	if !SlicesEqual(actual, expected) {
		t.Errorf("KubeAdm init args incorrect for master node. Expected %q, got %q", strings.Join(expected, " "), strings.Join(actual, " "))
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
			NetMgr:  nm,
			CNIArea: cniArea,
			Plugin:  "bridge",
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

	err := lazyjack.SetupForPlugin(n, c)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to create route")
	}
	expected := "unable to add pod network route for fd00:40:0:0:20::/80 to minion1: mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}
