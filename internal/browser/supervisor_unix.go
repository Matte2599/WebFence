//go:build !windows

package browser

import (
	"os/exec"
	"syscall"
)

func configureHelperProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func bindHelperProcess(cmd *exec.Cmd) (helperBoundary, error) {
	pid := cmd.Process.Pid
	return helperBoundary{
		kill: func() {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
		},
		close: func() {},
	}, nil
}
