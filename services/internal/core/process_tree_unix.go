//go:build !windows

package core

import (
	"os"
	"os/exec"
	"syscall"
)

type processTree struct{}

func prepareProcessTree(cmd *exec.Cmd) (*processTree, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processTree{}, nil
}

func (*processTree) attach(*os.Process) error { return nil }

func (*processTree) terminate(process *os.Process) error {
	if process == nil {
		return nil
	}
	if err := syscall.Kill(-process.Pid, syscall.SIGKILL); err != nil {
		return process.Kill()
	}
	return nil
}

func (*processTree) close() error { return nil }
