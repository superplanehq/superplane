package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// TestEchoHelloWorld builds fleet-manager and runner, starts one fleet process and one runner
// process, submits `echo 'hello world'`, and asserts on the completion webhook.
func TestEchoHelloWorld(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	binDir := t.TempDir()
	fleetPath := filepath.Join(binDir, "fleet-manager")
	runnerPath := filepath.Join(binDir, "worker")

	build := func(pkg, out string) {
		t.Helper()
		cmd := exec.Command("go", "build", "-o", out, pkg)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", pkg, err, out)
		}
	}
	build("./fleet-manager/cmd/fleet-manager", fleetPath)
	build("./runner/cmd/runner", runnerPath)

	dbPath := filepath.Join(t.TempDir(), "fleet-e2e.db")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	fleetCmd := exec.Command(fleetPath)
	fleetCmd.Env = append(os.Environ(),
		"DATABASE_PATH="+dbPath,
		"LISTEN_ADDR="+addr,
		"REAP_INTERVAL_SEC=3600",
	)
	fleetCmd.Stdout = io.Discard
	fleetCmd.Stderr = newTestLogWriter(t, "fleet-manager")
	if err := fleetCmd.Start(); err != nil {
		t.Fatalf("start fleet-manager: %v", err)
	}
	t.Cleanup(func() {
		_ = fleetCmd.Process.Signal(syscall.SIGTERM)
		_, _ = fleetCmd.Process.Wait()
	})

	baseURL := "http://" + addr
	waitReady(t, baseURL+"/healthz", 5*time.Second)

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

	runnerCmd := exec.Command(runnerPath)
	runnerCmd.Env = append(os.Environ(),
		"FLEET_MANAGER_URL="+baseURL,
		"RUNNER_ID=e2e-runner-1",
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

	body := bytes.NewBufferString(`{
  "command": ["sh", "-c", "echo 'hello world'"],
  "webhook_url": "` + whSrv.URL + `"
}`)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/tasks", body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create task status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var created api.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected task id")
	}

	var payload api.WebhookPayload
	select {
	case payload = <-received:
	case <-ctx.Done():
		t.Fatal("timed out waiting for webhook")
	}

	if payload.TaskID != created.ID {
		t.Errorf("webhook task_id: got %q want %q", payload.TaskID, created.ID)
	}
	if payload.Status != string(models.StatusSucceeded) {
		t.Errorf("status: got %q want %q", payload.Status, models.StatusSucceeded)
	}
	if payload.ExitCode != 0 {
		t.Errorf("exit_code: got %d want 0", payload.ExitCode)
	}
	if !strings.Contains(payload.Output, "hello world") {
		t.Errorf("output should contain hello world; got %q", payload.Output)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file is .../test/e2e_echo_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func waitReady(t *testing.T, healthURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, healthURL, nil)
		if err != nil {
			last = err.Error()
			time.Sleep(50 * time.Millisecond)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if err != nil {
			last = err.Error()
		} else {
			_ = resp.Body.Close()
			last = fmt.Sprintf("status %d", resp.StatusCode)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("health check never ready: %s", last)
}

type testLogWriter struct {
	t   *testing.T
	tag string
}

func newTestLogWriter(t *testing.T, tag string) io.Writer {
	return &testLogWriter{t: t, tag: tag}
}

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Helper()
	w.t.Logf("%s: %s", w.tag, strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}
