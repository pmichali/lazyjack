package orca

import (
	"bytes"
	"os/exec"

	"github.com/golang/glog"
)

func ContainerExists(c string) bool {
	cmd := exec.Command("docker", "inspect", c)
	output, _ := cmd.Output()
	if bytes.HasPrefix(output, []byte("[]")) {
		glog.V(4).Infof("No %q container", c)
		return false
	}
	glog.V(4).Infof("Container %q exists", c)
	return true
}
