//go:build linux || darwin || netbsd || freebsd || openbsd
// +build linux darwin netbsd freebsd openbsd

package pgroup

import (
	"os"
	"os/exec"
	"syscall"
)

func Setup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func Kill(cmd *exec.Cmd) {
	proc := cmd.Process
	if proc == nil {
		return
	}

	pgid, err := syscall.Getpgid(proc.Pid)
	if err != nil {
		syscall.Kill(-pgid, 15)
	}

	// just in case
	proc.Signal(os.Interrupt)
	proc.Signal(os.Kill)
}
