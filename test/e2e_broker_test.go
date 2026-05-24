package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// TestBrokerRoutesTask verifies task-broker queue → runner → caller webhook.
func TestBrokerRoutesTaskAndWebhook(t *testing.T) {
	skipIfPTYUnavailable(t)
	t.Parallel()

	root := moduleRoot(t)
	binDir := t.TempDir()
	runnerBin := filepath.Join(binDir, "runner")
	build := func(pkg, out string) {
		t.Helper()
		cmd := exec.Command("go", "build", "-o", out, pkg)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", pkg, err, out)
		}
	}
	build("./runner/cmd/runner", runnerBin)

	stack := startBrokerStack(t, root, binDir)
	httpClient := &http.Client{Timeout: 60 * time.Second}

	runnerCmd := exec.Command(runnerBin)
	runnerCmd.Env = runnerSubprocessEnv(
		"TASK_BROKER_URL="+stack.BrokerURL,
		"RUNNER_FLEET_ID="+stack.FleetID,
		"RUNNER_ID=e2e-broker-runner-1",
		"POLL_EMPTY_MS=20",
	)
	runnerCmd.Stdout = io.Discard
	runnerCmd.Stderr = newTestLogWriter(t, "runner")
	if err := runnerCmd.Start(); err != nil {
		t.Fatalf("start runner: %v", err)
	}
	t.Cleanup(func() {
		_ = runnerCmd.Process.Signal(syscall.SIGTERM)
		_, _ = runnerCmd.Process.Wait()
	})

	received := make(chan api.WebhookPayload, 1)
	whSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", http.StatusBadRequest)
			return
		}
		var p api.WebhookPayload
		if err := json.Unmarshal(b, &p); err != nil {
			http.Error(w, "json", http.StatusBadRequest)
			return
		}
		received <- p
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(whSrv.Close)

	createReq := api.BrokerCreateTaskRequest{
		CreateTaskRequest: api.CreateTaskRequest{
			Commands:   []string{`echo 'hello world'`, `echo second`},
			WebhookURL: whSrv.URL,
		},
		FleetID: stack.FleetID,
	}

	createPayload, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(waitCtx, http.MethodPost, stack.BrokerURL+"/v1/tasks", bytes.NewReader(createPayload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	setE2EBrokerAuth(req, stack.AuthToken)
	taskResp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer taskResp.Body.Close()
	taskBody, _ := io.ReadAll(taskResp.Body)
	if taskResp.StatusCode != http.StatusCreated {
		t.Fatalf("broker create task: status %d %s", taskResp.StatusCode, strings.TrimSpace(string(taskBody)))
	}
	var created api.BrokerCreateTaskResponse
	if err := json.Unmarshal(taskBody, &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("empty broker task id")
	}

	var payload api.WebhookPayload
	select {
	case payload = <-received:
	case <-waitCtx.Done():
		t.Fatal("timed out waiting for caller webhook")
	}

	if payload.TaskID != created.ID {
		t.Errorf("caller webhook task_id want broker id %q got %q", created.ID, payload.TaskID)
	}
	if payload.Status != string(models.StatusSucceeded) {
		t.Errorf("status: got %q", payload.Status)
	}
	if payload.ExitCode != 0 {
		t.Errorf("exit_code: got %d", payload.ExitCode)
	}

	pollURL := stack.BrokerURL + "/v1/tasks/" + created.ID
	pollReq, err := http.NewRequest(http.MethodGet, pollURL, nil)
	if err != nil {
		t.Fatalf("broker get task: %v", err)
	}
	setE2EBrokerAuth(pollReq, stack.AuthToken)
	pollResp, err := httpClient.Do(pollReq)
	if err != nil {
		t.Fatalf("broker get task: %v", err)
	}
	pollBody, _ := io.ReadAll(pollResp.Body)
	_ = pollResp.Body.Close()
	if pollResp.StatusCode != http.StatusOK {
		t.Fatalf("broker get task: status %d %s", pollResp.StatusCode, strings.TrimSpace(string(pollBody)))
	}
	var polled api.BrokerGetTaskResponse
	if err := json.Unmarshal(pollBody, &polled); err != nil {
		t.Fatal(err)
	}
	if polled.ID != created.ID {
		t.Errorf("poll id want %q got %q", created.ID, polled.ID)
	}
	if strings.ToLower(polled.Status) != string(models.StatusSucceeded) {
		t.Errorf("poll status want succeeded got %q", polled.Status)
	}
}
