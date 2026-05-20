package agent

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
)

func TestHostExecutorArgvUsesTaskEnvironment(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not on PATH: %v", err)
	}
	t.Setenv("RUNNER_TEST_ENV", "host-value")

	task := &api.TaskPayload{
		ID:          "task-env-argv",
		Command:     []string{sh, "-c", `printf %s "$RUNNER_TEST_ENV"`},
		Environment: []api.EnvironmentVariable{{Name: "RUNNER_TEST_ENV", Value: "task-value"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("host argv: exit=%d err=%v output=%q", exit, err, output)
	}
	if output != "task-value" {
		t.Fatalf("output = %q, want task-value", output)
	}
}

func TestHostExecutorCommandsUseTaskEnvironment(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")
	task := &api.TaskPayload{
		ID: "task-env-commands",
		Commands: []string{
			`printf "%s" "$RUNNER_TEST_ENV"`,
		},
		Environment: []api.EnvironmentVariable{{Name: "RUNNER_TEST_ENV", Value: "task-command-value"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{}).Execute(ctx, task, nil, "")
	if err != nil || exit != 0 {
		t.Fatalf("host commands: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "task-command-value") {
		t.Fatalf("output = %q, want task-command-value", output)
	}
}

// TestHostExecutorCommandsUseTaskEnvironmentWithResultFile runs on CI (pipe shell).
// Task env is applied after setResultEnv on that path; this guards result-file wiring end-to-end.
func TestHostExecutorCommandsUseTaskEnvironmentWithResultFile(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")
	task := &api.TaskPayload{
		ID: "task-env-pipe-result",
		Commands: []string{
			`printf "%s" "$RUNNER_TEST_ENV"`,
		},
		Environment: []api.EnvironmentVariable{{Name: "RUNNER_TEST_ENV", Value: "pipe-result-value"}},
	}
	resultPath := filepath.Join(t.TempDir(), "result.json")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{}).Execute(ctx, task, nil, resultPath)
	if err != nil || exit != 0 {
		t.Fatalf("host pipe commands: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "pipe-result-value") {
		t.Fatalf("output = %q, want pipe-result-value", output)
	}
}

// TestHostExecutorPTYCommandsUseTaskEnvironmentWithResultFile reproduces production:
// EC2 runners use PTY shell (default) and agent.execute always passes a result file path.
// Skipped when CI is set — Semaphore and similar builders often EIO on /dev/ptmx (see shell_unix_test.go).
func TestHostExecutorPTYCommandsUseTaskEnvironmentWithResultFile(t *testing.T) {
	skipPTYIntegrationOnCI(t)
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "0")

	task := &api.TaskPayload{
		ID: "task-env-pty",
		Commands: []string{
			`printf "%s" "$RUNNER_TEST_ENV"`,
		},
		Environment: []api.EnvironmentVariable{{Name: "RUNNER_TEST_ENV", Value: "pty-task-value"}},
	}
	resultPath := filepath.Join(t.TempDir(), "result.json")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exit, output, err := (&HostExecutor{}).Execute(ctx, task, nil, resultPath)
	if err != nil || exit != 0 {
		t.Fatalf("host pty commands: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "pty-task-value") {
		t.Fatalf("output = %q, want pty-task-value", output)
	}
}

func TestSetResultEnvPreservesExistingCmdEnv(t *testing.T) {
	t.Setenv("RUNNER_SETRESULTENV_PROBE", "from-os")
	cmd := exec.Command("true")
	cmd.Env = []string{"TASK_ONLY=1", "RUNNER_SETRESULTENV_PROBE=from-task"}
	setResultEnv(cmd, "/tmp/result.json")
	if len(cmd.Env) != 3 {
		t.Fatalf("cmd.Env len = %d, want 3: %#v", len(cmd.Env), cmd.Env)
	}
	if cmd.Env[0] != "TASK_ONLY=1" || cmd.Env[1] != "RUNNER_SETRESULTENV_PROBE=from-task" {
		t.Fatalf("task env overwritten: %#v", cmd.Env)
	}
	if !strings.HasPrefix(cmd.Env[2], envSuperplaneResultFile+"=") {
		t.Fatalf("missing result env: %#v", cmd.Env)
	}

	empty := exec.Command("true")
	setResultEnv(empty, "/tmp/result.json")
	if !strings.Contains(strings.Join(empty.Env, "\n"), "RUNNER_SETRESULTENV_PROBE=from-os") {
		t.Fatalf("expected os.Environ baseline when cmd.Env unset: %#v", empty.Env)
	}
}

func TestHostExecutorRejectsInvalidEnvironment(t *testing.T) {
	task := &api.TaskPayload{
		ID:          "task-invalid-env",
		Command:     []string{"true"},
		Environment: []api.EnvironmentVariable{{Name: "BAD-NAME", Value: "x"}},
	}

	exit, output, err := (&HostExecutor{}).Execute(context.Background(), task, nil, "")
	if err == nil {
		t.Fatalf("expected invalid environment error, exit=%d output=%q", exit, output)
	}
	if !strings.Contains(err.Error(), "environment variable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
