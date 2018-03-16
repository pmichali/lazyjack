package lazyjack_test

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		t.Errorf("FAILED: Expected to be able to read /etc/hostname: %s", err.Error())
	}

	_, err = lazyjack.GetFileContents("/etc/no-such-file")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to read non-existent file")
	}
	expected := "Unable to read /etc/no-such-file: open /etc/no-such-file: no such file or directory"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestSaveFileContents(t *testing.T) {
	path := TempFileName(os.TempDir(), "-area")
	err := os.MkdirAll(path, 0777)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure - unable to create temp area: %s", err.Error())
	} else {
		defer os.RemoveAll(path)
	}
	file := TempFileName(path, ".txt")
	backup := TempFileName(path, ".bak")
	err = ioutil.WriteFile(file, []byte("data"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure - unable to create temp file: %s", err.Error())
	}

	err = lazyjack.SaveFileContents([]byte("data"), file, backup)
	if err != nil {
		t.Fatalf("FAILED: Expected save to succeed, but it failed: %s", err.Error())
	}
}

func TestFailedSaveFileContents(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	file := TempFileName(basePath, ".txt")
	backup := os.TempDir() // Use directory as dest, so file rename fails
	err := ioutil.WriteFile(file, []byte("data"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure - unable to create temp file: %s", err.Error())
	}

	err = lazyjack.SaveFileContents([]byte("data"), file, backup)
	if err == nil {
		t.Fatalf("FAILED: Expected save to fail during backup of original file")
	}
	expected := "Unable to backup existing file"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestRecoverFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	file := TempFileName(basePath, ".txt")
	backup := TempFileName(basePath, ".bak")
	err := ioutil.WriteFile(backup, []byte("data"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure - unable to create temp file: %s", err.Error())
	}

	err = lazyjack.RecoverFile(file, backup, "unable to write file")
	if err == nil {
		t.Fatalf("FAILED: Expected recovery to fail")
	}
	expected := regexp.MustCompile("Unable to save updated .* [(]unable to write file[)], but restored from backup")
	if !expected.MatchString(err.Error()) {
		t.Fatalf("FAILED: Expected match to pattern %q, got %q", expected, err.Error())
	}
}

func TestFailedRecoverFile(t *testing.T) {
	basePath := TempFileName(os.TempDir(), "-area")
	HelperSetupArea(basePath, t)
	defer HelperCleanupArea(basePath, t)

	file := os.TempDir() // Use directory as dest, so file rename fails
	backup := TempFileName(basePath, ".txt")
	err := ioutil.WriteFile(backup, []byte("data"), 0777)
	if err != nil {
		t.Fatalf("ERROR: Test setup failure - unable to create temp file: %s", err.Error())
	}

	err = lazyjack.RecoverFile(file, backup, "unable to write file")
	if err == nil {
		t.Fatalf("FAILED: Expected recovery to fail")
	}
	expected := regexp.MustCompile("Unable to save updated /tmp [(]unable to write file[)] AND unable to restore backup file .* [(]rename .* /tmp: file exists[)]")
	if !expected.MatchString(err.Error()) {
		t.Fatalf("FAILED: Expected match to pattern %q, got %q", expected, err.Error())
	}
}
