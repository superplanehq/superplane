package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// TestBrokerRoutesTask verifies task-broker → fleet-manager → runner and that the caller webhook
// sees broker-scoped task_id plus fleet_task_id.
func TestBrokerRoutesTaskAndForwardsWebhook(t *testing.T) {
	skipIfPTYUnavailable(t)
	t.Parallel()

	root := moduleRoot(t)
	binDir := t.TempDir()
	fleetBin := filepath.Join(binDir, "fleet-manager")
	runnerBin := filepath.Join(binDir, "runner")
	brokerBin := filepath.Join(binDir, "task-broker")

	build := func(pkg, out string) {
		t.Helper()
		cmd := exec.Command("go", "build", "-o", out, pkg)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", pkg, err, out)
		}
	}
	build("./fleet-manager/cmd/fleet-manager", fleetBin)
	build("./runner/cmd/runner", runnerBin)
	build("./task-broker/cmd/task-broker", brokerBin)

	fleetDB := filepath.Join(t.TempDir(), "fleet.db")
	fleetAddr := freeTCPAddr(t)
	brokerDSN := os.Getenv("TEST_DATABASE_URL")
	if brokerDSN == "" {
		t.Skip("TEST_DATABASE_URL unset — required for task-broker e2e (see README)")
	}
	brokerAddr := freeTCPAddr(t)
	brokerPublic := "http://" + brokerAddr
	const brokerAuthToken = "e2e-broker-auth-token"

	fleetCmd := exec.Command(fleetBin)
	fleetCmd.Env = subprocessEnv(
		"DATABASE_PATH="+fleetDB,
		"LISTEN_ADDR="+fleetAddr,
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

	waitReady(t, "http://"+fleetAddr+"/healthz", 10*time.Second)

	brokerCmd := exec.Command(brokerBin)
	brokerCmd.Env = subprocessEnv(
		"DATABASE_URL="+brokerDSN,
		"LISTEN_ADDR="+brokerAddr,
		"BROKER_PUBLIC_URL="+brokerPublic,
		"AUTH_TOKEN="+brokerAuthToken,
	)
	brokerCmd.Stdout = io.Discard
	brokerCmd.Stderr = newTestLogWriter(t, "task-broker")
	if err := brokerCmd.Start(); err != nil {
		t.Fatalf("start task-broker: %v", err)
	}
	t.Cleanup(func() {
		_ = brokerCmd.Process.Signal(syscall.SIGTERM)
		_, _ = brokerCmd.Process.Wait()
	})

	waitReady(t, brokerPublic+"/healthz", 10*time.Second)

	httpClient := &http.Client{Timeout: 60 * time.Second}
	fleetBase := "http://" + fleetAddr

	regBody, err := json.Marshal(api.RegisterFleetRequest{
		ID:      "e2e-fleet",
		BaseURL: fleetBase,
		Labels:  []string{"e2e", "tier-test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	regReq, err := http.NewRequest(http.MethodPost, brokerPublic+"/v1/fleets", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	regReq.Header.Set("Content-Type", "application/json")
	regReq.Header.Set("Authorization", "Bearer "+brokerAuthToken)
	regResp, err := httpClient.Do(regReq)
	if err != nil {
		t.Fatal(err)
	}
	regRespBody, _ := io.ReadAll(regResp.Body)
	_ = regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register fleet: %s", strings.TrimSpace(string(regRespBody)))
	}

	runnerCmd := exec.Command(runnerBin)
	runnerCmd.Env = runnerSubprocessEnv(
		"FLEET_MANAGER_URL="+fleetBase,
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
		FleetLabels: []string{"e2e"},
	}

	createPayload, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(waitCtx, http.MethodPost, brokerPublic+"/v1/tasks", bytes.NewReader(createPayload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+brokerAuthToken)
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
	if payload.FleetTaskID == "" {
		t.Error("caller webhook fleet_task_id should be set")
	}
	if payload.Status != string(models.StatusSucceeded) {
		t.Errorf("status: got %q", payload.Status)
	}
	if payload.ExitCode != 0 {
		t.Errorf("exit_code: got %d", payload.ExitCode)
	}

	pollURL := brokerPublic + "/v1/tasks/" + created.ID
	pollReq, err := http.NewRequest(http.MethodGet, pollURL, nil)
	if err != nil {
		t.Fatalf("broker get task: %v", err)
	}
	pollReq.Header.Set("Authorization", "Bearer "+brokerAuthToken)
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
	if polled.TaskID != created.ID {
		t.Errorf("poll task_id want %q got %q", created.ID, polled.TaskID)
	}
	if strings.ToLower(polled.Status) != string(models.StatusSucceeded) {
		t.Errorf("poll status want succeeded got %q", polled.Status)
	}
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}
