//go:build unix

package agent

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func prepareTaskProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killTaskProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		pgid = cmd.Process.Pid
	}
	if pgid <= 0 || pgid == currentProcessGroup() {
		_ = cmd.Process.Kill()
		return
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err == nil {
		time.Sleep(100 * time.Millisecond)
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		_ = cmd.Process.Kill()
	}
}

func currentProcessGroup() int {
	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return -1
	}
	return pgid
}
