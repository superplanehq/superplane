//go:build unix

package agent

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

func skipIfPTYUnavailable(t *testing.T) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "--noediting", "-i")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	f, err := pty.Start(cmd)
	if err != nil {
		t.Skipf("PTY required (not available here): %v", err)
	}
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	_ = f.Close()
	_ = cmd.Wait()
}

func TestHostShellDirectivesEcho(t *testing.T) {
	skipIfPTYUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	code, out, err := runHostShellDirectives(ctx, 128*1024, []string{`echo 'hello'`})
	t.Logf("code=%d out=%q err=%v", code, out, err)
	if err != nil || code != 0 {
		t.Fatalf("want success, got code=%d err=%v out=%q", code, err, out)
	}
	if out == "" || !strings.Contains(out, "hello") {
		t.Fatalf("expected hello in output: %q", out)
	}
}
