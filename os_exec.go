package lazyjack

import (
	"fmt"
	"os/exec"

	"github.com/golang/glog"

	"strings"
)

// DoExecCommand performs an OS command, returning the output and
// results of running the command. Used to invoke KubeAdm commands.
func DoExecCommand(cmd string, args []string) (string, error) {
	a := strings.Join(args, " ")
	glog.V(4).Infof("Invoking: %s %s", cmd, a)
	c := exec.Command(cmd, args...)
	output, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("failed running %q with args %q: %v (%s)", cmd, a, err, output)
	}
	glog.V(4).Infof("Command %q with args %q successful", cmd, a)
	return string(output), nil
}
