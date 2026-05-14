package e2e_test

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/creack/pty"
)

// subprocessEnv builds an environment for child processes spawned in tests.
// It drops AUTH_TOKEN so a developer shell exporting fleet-manager AUTH_TOKEN
// does not make unauthenticated POST /v1/tasks return 401.
func subprocessEnv(extra ...string) []string {
	var out []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "AUTH_TOKEN=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, extra...)
}

// runnerSubprocessEnv is like subprocessEnv but adjusts the worker for CI builders where
// pty.Start() succeeds yet reads on /dev/ptmx fail later (EIO), so tasks never finish and
// webhooks time out. The pipe bundle path avoids a PTY while still exercising bash directives.
//
// E2e always uses RUNNER_TRANSPORT=http (POST claim/complete): these tests focus on task +
// webhook behavior; WebSocket is covered in fleet-manager and runner unit tests. We also
// strip any inherited RUNNER_TRANSPORT because on Linux the first duplicate env key wins,
// so a developer shell exporting RUNNER_TRANSPORT=websocket would otherwise override a
// trailing assignment.
func runnerSubprocessEnv(extra ...string) []string {
	env := subprocessEnv(extra...)
	var out []string
	for _, kv := range env {
		if strings.HasPrefix(kv, "RUNNER_TRANSPORT=") {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "RUNNER_TRANSPORT=http")
	if os.Getenv("CI") != "" {
		out = append(out, "RUNNER_SHELL_USE_PIPE=1")
	}
	return out
}

// skipIfPTYUnavailable skips the test when Bash+PTY cannot run (e.g. hardened
// sandboxes). Production runners require PTY; this only relaxes local/CI preflight.
func skipIfPTYUnavailable(t *testing.T) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	cmd := exec.Command(bash, "--norc", "--noprofile", "+m", "-i")
	// Do not set Setpgid: creack/pty uses Setsid+Setctty; combining them yields fork/exec EPERM on Darwin/Linux.
	f, err := pty.Start(cmd)
	if err != nil {
		t.Skipf("PTY required for runner e2e (not available here): %v", err)
	}
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	_ = f.Close()
	_ = cmd.Wait()
}
