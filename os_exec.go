package lazyjack

import (
	"fmt"
	"os/exec"

	"github.com/golang/glog"

	"strings"
)

func DoExecCommand(cmd string, args []string) (string, error) {
	a := strings.Join(args, " ")
	glog.V(4).Infof("Invoking: %s %s", cmd, a)
	c := exec.Command(cmd, args...)
	output, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("Failed running %q with args %q: %s (%s)", cmd, a, err.Error(), output)
	}
	glog.V(4).Infof("Command %q with args %q successful", cmd, a)
	return string(output), nil
}
