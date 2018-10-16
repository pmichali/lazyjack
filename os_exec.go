package lazyjack

import (
	"fmt"
	"os/exec"

	"github.com/golang/glog"

	"strings"
)

// ExecCommandFuncType is a function for performing OS commands
type ExecCommandFuncType = func(string, []string) (string, error)

var execCommand ExecCommandFuncType = OsExecCommand

// RegisterExecCommand will register a OS command function for exec calls.
// Used ONLY by unit test.
func RegisterExecCommand(cmdFunc ExecCommandFuncType) {
	execCommand = cmdFunc
}

// DoExecCommand is a wrapper for performing OS commands and returning
// output or error. Can be overridden for unit tests.
func DoExecCommand(cmd string, args []string) (string, error) {
	return execCommand(cmd, args)
}

func OsExecCommand(cmd string, args []string) (string, error) {
	a := strings.Join(args, " ")
	glog.V(4).Infof("Invoking: %s %s", cmd, a)
	c := exec.Command(cmd, args...)
	output, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed running %q with args %q: %v. Output=%q", cmd, a, err, output)
	}
	glog.V(4).Infof("Command %q with args %q successful", cmd, a)
	return string(output), nil
}
