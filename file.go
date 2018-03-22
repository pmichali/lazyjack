package lazyjack

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
)

func GetFileContents(file string) ([]byte, error) {
	glog.V(4).Infof("Reading %s contents", file)
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("unable to read %s: %v", file, err)
	}
	return contents, err
}

func SaveFileContents(contents []byte, file, backup string) error {
	glog.V(4).Infof("Saving updated %s", file)
	_, err := os.Stat(file)
	exists := true
	if os.IsNotExist(err) {
		exists = false
	}
	if exists {
		err := os.Rename(file, backup)
		if err != nil {
			return fmt.Errorf("unable to backup existing file %s to %s: %v", file, backup, err)
		}
		glog.V(4).Infof("Backed up existing %s to %s", file, backup)
	}
	err = ioutil.WriteFile(file, contents, 0755)
	if err != nil {
		if exists {
			return RecoverFile(file, backup, err.Error())
		}
		return fmt.Errorf("unable to save %s", file)
	}
	glog.V(4).Infof("Saved %s", file)
	return nil
}

func RecoverFile(file, backup, saveErr string) error {
	err := os.Rename(backup, file)
	if err != nil {
		return fmt.Errorf("unable to save updated %s (%s) AND unable to restore backup file %s (%v)",
			file, saveErr, backup, err)
	}
	return fmt.Errorf("unable to save updated %s (%s), but restored from backup", file, saveErr)
}
