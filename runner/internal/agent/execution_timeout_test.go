package agent

import (
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
