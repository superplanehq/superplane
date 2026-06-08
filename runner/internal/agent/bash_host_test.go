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

func TestHostExecutorBashWithSetupCommands(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")

	resultPath := filepath.Join(t.TempDir(), "result.json")
	workDir := t.TempDir()
	task := &api.TaskPayload{
		ID:      "task-bash-setup",
		RunMode: string(models.RunModeBash),
		SetupCommands: []string{
			`echo setup-ran > setup-marker.txt`,
		},
		Script: `main() {
  local marker
  marker=$(tr -d '\n' < setup-marker.txt)
  echo "{\"marker\": \"$marker\"}"
}`,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, out, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, resultPath)
	if err != nil || exit != 0 {
		t.Fatalf("bash host with setup: exit=%d err=%v out=%q", exit, err, out)
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

func TestHostExecutorBashSetupFailureSkipsScript(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}
	t.Setenv("RUNNER_SHELL_USE_PIPE", "1")

	resultPath := filepath.Join(t.TempDir(), "result.json")
	task := &api.TaskPayload{
		ID:      "task-bash-setup-fail",
		RunMode: string(models.RunModeBash),
		SetupCommands: []string{
			"false",
		},
		Script: `main() {
  echo '{"should":"not-run"}'
}`,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, _, err := (&HostExecutor{TaskWorkDir: t.TempDir()}).Execute(ctx, task, nil, resultPath)
	if exit == 0 {
		t.Fatalf("exit = %d, want non-zero", exit)
	}
	_ = err
}

func TestHostExecutorBashScript(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not on PATH: %v", err)
	}

	resultPath := filepath.Join(t.TempDir(), "result.json")
	task := &api.TaskPayload{
		ID:      "task-bash-host",
		RunMode: string(models.RunModeBash),
		Script: `main() {
  local payload="$1"
  local num
  num=$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["GitHub PR"]["data"]["number"])' "$payload")
  echo "{\"pr\": $num, \"ok\": true}"
}`,
		MessageChain: json.RawMessage(`{"GitHub PR":{"data":{"number":99}}}`),
	}
	workDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exit, out, err := (&HostExecutor{TaskWorkDir: workDir}).Execute(ctx, task, nil, resultPath)
	if err != nil || exit != 0 {
		t.Fatalf("bash host: exit=%d err=%v out=%q", exit, err, out)
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
