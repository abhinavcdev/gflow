package cmd

import (
	"os/exec"
)

// newExecCommand creates an exec.Cmd
func newExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
