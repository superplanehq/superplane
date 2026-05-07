//go:build unix

package agent

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func skipIfPTYUnavailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	// Semaphore/Docker agents often allow opening /dev/ptmx but fail on read (EIO). Probe the full
	// runHostShellDirectives path instead of only pty.Start.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	code, _, err := runHostShellDirectives(ctx, 8*1024, []string{`echo 'pty-probe'`})
	if err != nil || code != 0 {
		t.Skipf("host PTY shell not available in this environment (code=%d): %v", code, err)
	}
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
