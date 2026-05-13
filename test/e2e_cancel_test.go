package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

func TestCancelQueuedImmediateWebhook(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	binDir := t.TempDir()
	fleetPath := filepath.Join(binDir, "fleet-manager")
	cmd := exec.Command("go", "build", "-o", fleetPath, "./fleet-manager/cmd/fleet-manager")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build fleet-manager: %v\n%s", err, out)
	}

	dbPath := filepath.Join(t.TempDir(), "fleet-cancel-queued.db")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	fleetCmd := exec.Command(fleetPath)
	fleetCmd.Env = subprocessEnv(
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

	createBody, err := json.Marshal(api.CreateTaskRequest{
		Command:    []string{"echo", "never-runs"},
		WebhookURL: whSrv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/tasks", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create: %d %s", resp.StatusCode, b)
	}
	var created api.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	cancelReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/tasks/"+created.ID+"/cancel", nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatal(err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(cancelResp.Body)
		t.Fatalf("cancel: %d %s", cancelResp.StatusCode, b)
	}
	var cancelOut api.CancelTaskResponse
	if err := json.NewDecoder(cancelResp.Body).Decode(&cancelOut); err != nil {
		t.Fatal(err)
	}
	if cancelOut.State != "canceled" {
		t.Fatalf("cancel state: %+v", cancelOut)
	}

	var payload api.WebhookPayload
	select {
	case payload = <-received:
	case <-ctx.Done():
		t.Fatal("timed out waiting for webhook")
	}
	if payload.Status != string(models.StatusCanceled) {
		t.Fatalf("webhook status: %q", payload.Status)
	}
	if payload.ExitCode != 130 {
		t.Fatalf("webhook exit: %d", payload.ExitCode)
	}
}

func TestCancelClaimedStopsArgvSleep(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)
	binDir := t.TempDir()
	fleetPath := filepath.Join(binDir, "fleet-manager")
	runnerPath := filepath.Join(binDir, "worker")
	for _, b := range []struct {
		pkg string
		out string
	}{
		{"./fleet-manager/cmd/fleet-manager", fleetPath},
		{"./runner/cmd/runner", runnerPath},
	} {
		cmd := exec.Command("go", "build", "-o", b.out, b.pkg)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", b.pkg, err, out)
		}
	}

	dbPath := filepath.Join(t.TempDir(), "fleet-cancel-mid.db")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	fleetCmd := exec.Command(fleetPath)
	fleetCmd.Env = subprocessEnv(
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
	runnerCmd.Env = runnerSubprocessEnv(
		"FLEET_MANAGER_URL="+baseURL,
		"RUNNER_ID=e2e-cancel-runner",
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

	createBody, err := json.Marshal(api.CreateTaskRequest{
		Command:    []string{"sleep", "120"},
		WebhookURL: whSrv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/tasks", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create: %d %s", resp.StatusCode, b)
	}
	var created api.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	time.Sleep(400 * time.Millisecond)

	cancelReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/tasks/"+created.ID+"/cancel", nil)
	if err != nil {
		t.Fatal(err)
	}
	cancelResp, err := http.DefaultClient.Do(cancelReq)
	if err != nil {
		t.Fatal(err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(cancelResp.Body)
		t.Fatalf("cancel: %d %s", cancelResp.StatusCode, b)
	}
	var cancelOut api.CancelTaskResponse
	if err := json.NewDecoder(cancelResp.Body).Decode(&cancelOut); err != nil {
		t.Fatal(err)
	}
	if cancelOut.State != "cancel_requested" {
		t.Fatalf("cancel API state: %+v", cancelOut)
	}

	var payload api.WebhookPayload
	select {
	case payload = <-received:
	case <-ctx.Done():
		t.Fatal("timed out waiting for canceled webhook")
	}
	if payload.Status != string(models.StatusCanceled) {
		t.Fatalf("webhook status: %q output=%q", payload.Status, payload.Output)
	}
}
