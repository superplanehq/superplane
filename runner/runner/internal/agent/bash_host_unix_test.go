//go:build !windows

package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

func TestHostExecutorBashFailureKillsBackgroundChild(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}

	tmp := t.TempDir()
	pidPath := filepath.Join(tmp, "child.pid")
	resultPath := filepath.Join(tmp, "result.json")
	task := &api.TaskPayload{
		ID:      "task-bash-kills-child",
		RunMode: string(models.RunModeBash),
		Script: fmt.Sprintf(`nohup sleep 60 >/dev/null 2>&1 &
echo $! > %s
exit 1
`, shellQuote(pidPath)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, _, _ := (&HostExecutor{TaskWorkDir: tmp}).Execute(ctx, task, nil, resultPath)
	if exit == 0 {
		t.Fatal("expected bash task to fail")
	}
	pidBytes, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		t.Fatal(err)
	}
	if !waitProcessGone(pid, 2*time.Second) {
		t.Fatalf("background child pid %d is still alive after failed task", pid)
	}
}

func shellQuote(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}

func waitProcessGone(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !processExists(pid) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func processExists(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
