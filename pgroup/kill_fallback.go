// +build !windows,!linux,!darwin,!netbsd,!freebsd,!openbsd

package pgroup

import (
	"os"
	"os/exec"
)

func Setup(c *exec.Cmd) {}

func Kill(cmd *exec.Cmd) {
	proc := cmd.Process
	if proc == nil {
		return
	}
	proc.Signal(os.Interrupt)
	proc.Signal(os.Kill)
}
