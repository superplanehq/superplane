package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
)

func TestExecutionWallDuration_defaultAndCap(t *testing.T) {
	cfgNoCap := DefaultConfig()
	if got := executionWallDuration(cfgNoCap, &api.TaskPayload{}); got != time.Duration(api.DefaultExecutionTimeoutSeconds)*time.Second {
		t.Fatalf("default without cap: got %v want 9m", got)
	}

	cfg := DefaultConfig()
	cfg.MaxExecutionSeconds = 60
	if got := executionWallDuration(cfg, &api.TaskPayload{}); got != 60*time.Second {
		t.Fatalf("default with cap: got %v want 60s", got)
	}

	v := 120
	task2 := &api.TaskPayload{ExecutionTimeoutSeconds: &v}
	if got := executionWallDuration(cfg, task2); got != 60*time.Second {
		t.Fatalf("capped: got %v want 60s", got)
	}

	v30 := 30
	task3 := &api.TaskPayload{ExecutionTimeoutSeconds: &v30}
	if got := executionWallDuration(cfg, task3); got != 30*time.Second {
		t.Fatalf("under cap: got %v want 30s", got)
	}
}

func TestRunnerFailureKindParentCancellationIsRunnerInfra(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	execCtx, cancelExec := context.WithCancel(context.Background())
	cancelExec()

	got := runnerFailureKind(parent, execCtx, context.Canceled)
	if got != api.FailureKindRunnerInfra {
		t.Fatalf("failure kind: got %q want %q", got, api.FailureKindRunnerInfra)
	}
}

func TestRunnerFailureKindTimeoutIsNotRunnerInfra(t *testing.T) {
	parent := context.Background()
	execCtx, cancelExec := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancelExec()
	<-execCtx.Done()

	got := runnerFailureKind(parent, execCtx, context.DeadlineExceeded)
	if got != "" {
		t.Fatalf("failure kind: got %q want empty", got)
	}
}

func TestRunnerFailureKindScriptFailureIsNotRunnerInfra(t *testing.T) {
	parent := context.Background()
	execCtx := context.Background()

	got := runnerFailureKind(parent, execCtx, errors.New("script failed"))
	if got != "" {
		t.Fatalf("failure kind: got %q want empty", got)
	}
}
