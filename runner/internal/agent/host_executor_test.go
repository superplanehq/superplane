package agent

import (
	"context"
	"os/exec"
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

	exit, output, err := (&HostExecutor{}).Execute(ctx, task, nil)
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

	exit, output, err := (&HostExecutor{}).Execute(ctx, task, nil)
	if err != nil || exit != 0 {
		t.Fatalf("host commands: exit=%d err=%v output=%q", exit, err, output)
	}
	if !strings.Contains(output, "task-command-value") {
		t.Fatalf("output = %q, want task-command-value", output)
	}
}

func TestHostExecutorRejectsInvalidEnvironment(t *testing.T) {
	task := &api.TaskPayload{
		ID:          "task-invalid-env",
		Command:     []string{"true"},
		Environment: []api.EnvironmentVariable{{Name: "BAD-NAME", Value: "x"}},
	}

	exit, output, err := (&HostExecutor{}).Execute(context.Background(), task, nil)
	if err == nil {
		t.Fatalf("expected invalid environment error, exit=%d output=%q", exit, output)
	}
	if !strings.Contains(err.Error(), "environment variable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
