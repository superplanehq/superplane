package agent

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

func TestHostExecutorJavaScriptWithSetupCommands(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")

	resultPath := filepath.Join(t.TempDir(), "result.json")
	workDir := t.TempDir()
	task := &api.TaskPayload{
		ID:      "task-js-setup",
		RunMode: string(models.RunModeJavaScript),
		SetupCommands: []string{
			`echo setup-ran > setup-marker.txt`,
		},
		Script: `function main() {
  return { marker: require('fs').readFileSync("setup-marker.txt", "utf8").trim() };
}`,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, out, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, resultPath)
	if err != nil || exit != 0 {
		t.Fatalf("javascript host with setup: exit=%d err=%v out=%q", exit, err, out)
	}
	b, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Marker string `json:"marker"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("result json: %v body=%s", err, b)
	}
	if result.Marker != "setup-ran" {
		t.Fatalf("result = %+v", result)
	}
}

func TestHostExecutorJavaScriptSetupFailureSkipsScript(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")

	resultPath := filepath.Join(t.TempDir(), "result.json")
	task := &api.TaskPayload{
		ID:      "task-js-setup-fail",
		RunMode: string(models.RunModeJavaScript),
		SetupCommands: []string{
			"false",
		},
		Script: `function main() { return { should: "not-run" }; }`,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, _, err := (&HostExecutor{TaskWorkDir: t.TempDir()}).Execute(ctx, task, nil, resultPath)
	if exit == 0 {
		t.Fatalf("exit = %d, want non-zero", exit)
	}
	if _, statErr := os.Stat(resultPath); statErr == nil {
		b, _ := os.ReadFile(resultPath)
		if len(b) > 0 {
			t.Fatalf("expected empty/missing result file after setup failure, got %s", b)
		}
	}
	_ = err
}

func TestHostExecutorJavaScriptScript(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skipf("node not on PATH: %v", err)
	}

	resultPath := filepath.Join(t.TempDir(), "result.json")
	task := &api.TaskPayload{
		ID:      "task-js-host",
		RunMode: string(models.RunModeJavaScript),
		Script: `function main() {
  return { pr: $['GitHub PR'].data.number, ok: true };
}`,
		MessageChain: json.RawMessage(`{"GitHub PR":{"data":{"number":99}}}`),
	}
	workDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, _, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, resultPath)
	if err != nil || exit != 0 {
		t.Fatalf("javascript host: exit=%d err=%v", exit, err)
	}
	b, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		PR int  `json:"pr"`
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("result json: %v body=%s", err, b)
	}
	if result.PR != 99 || !result.OK {
		t.Fatalf("result = %+v", result)
	}
}
