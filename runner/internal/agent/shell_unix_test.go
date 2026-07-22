//go:build unix

package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

func skipPTYIntegrationOnCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("PTY tests need a working pseudo-terminal; skipped when CI is set (often EIO on /dev/ptmx)")
	}
}

// TestHostShellDirectivesEcho exercises the default production path: Bash + PTY + marker protocol.
// It requires a working PTY (typical Linux runners and normal macOS terminals). Sandboxed IDEs
// often break PTY reads; many CI builders return EIO on /dev/ptmx — skipped when env CI is set.
// Run TestHostShellPipeBundleEcho locally or on hosts with a real PTY.
// TestPTYProbeMinimal mirrors cmd/ptyprobe (bash -i + creack/pty + substring marker check).
// TestHostShellDirectivesEcho covers the real runner protocol (exact boot line + directives).
func TestRunShellPTYSessionEcho(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("bash required: %v", err)
	}
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "-i")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	code, out, err := runShellPTYSession(ctx, 128*1024, cmd, directivesFromStrings([]string{`echo 'hello'`}), nil, "")
	t.Logf("code=%d out=%q err=%v", code, out, err)
	if err != nil || code != 0 {
		t.Fatalf("runShellPTYSession: code=%d err=%v out=%q", code, err, out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected hello: %q", out)
	}
}

func TestPTYProbeMinimal(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("bash required: %v", err)
	}
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "-i")
	m, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty.Start: %v", err)
	}
	defer func() { _ = m.Close() }()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	marker := fmt.Sprintf("bootready-%d", time.Now().UnixNano())
	if _, err := m.Write([]byte(fmt.Sprintf("echo '%s'\n", marker))); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, 4096)
	var acc []byte
	deadline := time.After(8 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout acc=%q", acc)
		default:
		}
		n, rerr := m.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			if bytes.Contains(acc, []byte(marker)) {
				return
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				t.Fatalf("EOF acc=%q", acc)
			}
			t.Fatalf("read: %v acc=%q", rerr, acc)
		}
	}
}

func TestHostShellDirectivesEcho(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("bash required: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	code, out, err := runHostShellDirectives(ctx, 128*1024, t.TempDir(), []string{`echo 'hello'`}, nil, nil, "")
	t.Logf("code=%d out=%q err=%v", code, out, err)
	if err != nil || code != 0 {
		t.Fatalf("pty path (production default) failed: code=%d err=%v out=%q", code, err, out)
	}
	if out == "" || !strings.Contains(out, "hello") {
		t.Fatalf("expected hello in output: %q", out)
	}
}

func TestHostShellDirectivesExitAliasKeepsShell(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("bash required: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dir := t.TempDir()
	work := filepath.Join(dir, "work")
	code, out, err := runHostShellDirectives(ctx, 128*1024, dir, []string{
		fmt.Sprintf("mkdir -p %s; cd %s; export RUNNER_EXIT_MARK=kept; echo hello; exit 1; echo there", work, work),
	}, nil, nil, "")
	t.Logf("code=%d out=%q err=%v", code, out, err)

	// Session-boot alias makes top-level exit→return: clean status, shell stays up.
	if code != 1 {
		t.Fatalf("expected exit code 1, got code=%d err=%v out=%q", code, err, out)
	}
	if err == nil || !strings.Contains(err.Error(), "exit code") {
		t.Fatalf("expected exit-code error from end marker, got %v", err)
	}
	if strings.Contains(err.Error(), "shell closed") {
		t.Fatalf("exit killed the PTY shell: %v", err)
	}
	if strings.Contains(out, "there") {
		t.Fatalf("expected fail-fast before echo there; out=%q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected hello before exit; out=%q", out)
	}
}

func TestHostShellDirectivesExitAliasPersistsAcrossCommands(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("bash required: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dir := t.TempDir()
	work := filepath.Join(dir, "work")
	// Alias is installed once at session boot; a later command must still see exit→return,
	// and cd/export from a prior successful command must persist.
	code, out, err := runHostShellDirectives(ctx, 128*1024, dir, []string{
		fmt.Sprintf("mkdir -p %s; cd %s; export RUNNER_EXIT_MARK=kept; exit 0", work, work),
		`printf 'mark=%s cwd=%s\n' "$RUNNER_EXIT_MARK" "$(pwd -P)"; echo hello; exit 1; echo there`,
	}, nil, nil, "")
	t.Logf("code=%d out=%q err=%v", code, out, err)
	if code != 1 {
		t.Fatalf("expected second-command exit code 1, got code=%d err=%v out=%q", code, err, out)
	}
	if strings.Contains(err.Error(), "shell closed") {
		t.Fatalf("exit on later command killed the PTY shell: %v", err)
	}
	if !strings.Contains(out, "mark=kept") || !strings.Contains(out, "/work") {
		t.Fatalf("cd/export should persist across commands; out=%q", out)
	}
	if strings.Contains(out, "there") {
		t.Fatalf("expected aliased exit to stop second command; out=%q", out)
	}
}

func TestEndMarkerStreamHoldback(t *testing.T) {
	t.Parallel()
	endMark := "e-0123456789abcdef-0123456789abcdef"
	tmpl := "\x01 " + endMark + " 0\n"

	tests := []struct {
		name string
		data string
		want int
	}{
		{name: "empty", data: "", want: 0},
		{name: "no marker prefix", data: "hello world\n", want: 0},
		{name: "partial SOH", data: "out\n\x01", want: 1},
		{name: "partial marker line", data: "out\n\x01 " + endMark[:8], want: len("\x01 " + endMark[:8])},
		{name: "complete marker withheld", data: "out\n" + tmpl, want: len(tmpl)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := endMarkerStreamHoldback([]byte(tc.data), endMark)
			if got != tc.want {
				t.Fatalf("holdback=%d want=%d data=%q", got, tc.want, tc.data)
			}
		})
	}
}

type syncLiveWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *syncLiveWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncLiveWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *syncLiveWriter) waitContains(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(w.String(), substr) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func TestRunShellPTYSessionLiveStreamsIncrementally(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("bash required: %v", err)
	}
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "-i")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	live := &syncLiveWriter{}
	done := make(chan struct{})
	var code int
	var runErr error
	go func() {
		code, _, runErr = runShellPTYSession(ctx, 128*1024, cmd, directivesFromStrings([]string{
			`for i in 1 2 3; do echo line$i; sleep 0.15; done`,
		}), live, "")
		close(done)
	}()

	if !live.waitContains("line1", 2*time.Second) {
		t.Fatalf("live log did not receive line1 during command; live=%q", live.String())
	}
	if strings.Contains(live.String(), "line3") {
		t.Fatalf("expected line3 only after later streaming; live=%q", live.String())
	}
	<-done
	if runErr != nil || code != 0 {
		t.Fatalf("runShellPTYSession: code=%d err=%v live=%q", code, runErr, live.String())
	}
	if !strings.Contains(live.String(), "line3") {
		t.Fatalf("expected all lines in live log; live=%q", live.String())
	}
}

// TestHostShellPipeBundleEcho covers RUNNER_SHELL_USE_PIPE only: non-PTY bundle execution.
// Production EC2 workers should not set that env; this is an explicit escape hatch + regression test.
func TestHostShellPipeBundleEcho(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("bash required: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	code, out, err := runHostShellDirectives(ctx, 128*1024, t.TempDir(), []string{`echo 'hello'`}, nil, nil, "")
	t.Logf("code=%d out=%q err=%v", code, out, err)
	if err != nil || code != 0 {
		t.Fatalf("pipe bundle path failed: code=%d err=%v out=%q", code, err, out)
	}
	if out == "" || !strings.Contains(out, "hello") {
		t.Fatalf("expected hello in output: %q", out)
	}
}
