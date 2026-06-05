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
