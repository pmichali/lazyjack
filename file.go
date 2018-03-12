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
		err = fmt.Errorf("Unable to read %s: %s", file, err.Error())
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
			return fmt.Errorf("Unable to backup existing file %s to %s: %s", file, backup, err.Error())
		} else {
			glog.V(4).Infof("Backed up existing %s to %s", file, backup)
		}
	}
	err = ioutil.WriteFile(file, contents, 0755)
	if err != nil {
		if exists {
			err2 := os.Rename(backup, file)
			if err2 != nil {
				return fmt.Errorf("Unable to save updated %s (%s) AND unable to restore backup file %s (%s)",
					file, err.Error(), backup, err2.Error())
			}
			return fmt.Errorf("Unable to save updated %s (%s), but restored from backup", file, err.Error())
		} else {
			return fmt.Errorf("Unable to save %s", file)
		}
	}
	glog.V(4).Infof("Saved %s", file)
	return nil
}
