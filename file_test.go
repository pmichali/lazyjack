package lazyjack_test

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TempFileName(area, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(area, "test"+hex.EncodeToString(randBytes)+suffix)
}

func HelperSetupArea(basePath string, t *testing.T) {
	err := os.MkdirAll(basePath, 0700)
	if err != nil {
		t.Fatalf("ERROR: unable to setup area %q for test: %s", basePath, err.Error())
	}
}

func HelperCleanupArea(basePath string, t *testing.T) {
	err := os.RemoveAll(basePath)
	if err != nil {
		t.Fatalf("ERROR: Test cleanup failure: %s", err.Error())
	}
}

func HelperMakeReadOnly(basePath string, t *testing.T) {
	err := os.Chmod(basePath, 0400)
	if err != nil {
		t.Fatalf("ERROR: Unable to make area read only for test")
	}
}

func HelperMakeWriteable(basePath string, t *testing.T) {
	err := os.Chmod(basePath, 0777)
	if err != nil {
		t.Fatalf("ERROR: Unable to make area writeable for test cleanup")
	}
}

func TestGetFileContents(t *testing.T) {
	_, err := lazyjack.GetFileContents("/etc/hostname")
	if err != nil {
		t.Errorf("Expected to be able to read /etc/hostname: %s", err.Error())
	}

	_, err = lazyjack.GetFileContents("/etc/no-such-file")
	if err == nil {
		t.Errorf("Expected to not be able to read non-existent file")
	}
}

func TestSaveFileContents(t *testing.T) {
	path := TempFileName(os.TempDir(), "-area")
	err := os.MkdirAll(path, 0777)
	if err != nil {
		t.Fatalf("Test setup failure - unable to create temp area: %s", err.Error())
	} else {
		defer os.RemoveAll(path)
	}
	file := TempFileName(path, ".txt")
	backup := TempFileName(path, ".bak")
	err = ioutil.WriteFile(file, []byte("data"), 0777)
	if err != nil {
		t.Fatalf("Test setup failure - unable to create temp file: %s", err.Error())
	}

	// Test normal
	err = lazyjack.SaveFileContents([]byte("data"), file, backup)
	if err != nil {
		t.Fatalf("Expected save to succeed, but it failed: %s", err.Error())
	}

	// Backup failed (cannot rename to a directory)
	err = lazyjack.SaveFileContents([]byte("data"), file, os.TempDir())
	if err == nil {
		t.Fatalf("Expected save to fail when backup file is bad - but it worked")
	}
}
