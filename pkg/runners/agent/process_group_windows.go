//go:build windows

package agent

import "os/exec"

func prepareTaskProcessGroup(_ *exec.Cmd) {}

func killTaskProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
