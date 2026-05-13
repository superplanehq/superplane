package fleetmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/wsrunner"
)

func TestRunnerStream_HappyPath(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "fleet.db")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	hub := NewWaitHub()
	srv := &Server{
		Store:      st,
		Webhook:    nil,
		Log:        slog.Default(),
		TaskNotify: hub,
	}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{}))
	defer ts.Close()

	body := `{"commands":["echo hello"],"webhook_url":"https://example.com/hook"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create task: %d", resp.StatusCode)
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/runners/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	hello := wsrunner.Hello{Type: wsrunner.TypeHello, RunnerID: "runner-ws-1", LeaseSeconds: 300}
	if err := conn.WriteJSON(hello); err != nil {
		t.Fatal(err)
	}

	var taskMsg wsrunner.Task
	if err := conn.ReadJSON(&taskMsg); err != nil {
		t.Fatal(err)
	}
	if taskMsg.Type != wsrunner.TypeTask || taskMsg.Task == nil || taskMsg.Task.ID == "" {
		t.Fatalf("task payload: %+v", taskMsg)
	}

	comp := wsrunner.Complete{
		Type:     wsrunner.TypeComplete,
		TaskID:   taskMsg.Task.ID,
		RunnerID: "runner-ws-1",
		ExitCode: 0,
		Output:   "out\n",
	}
	if err := conn.WriteJSON(comp); err != nil {
		t.Fatal(err)
	}

	var ack wsrunner.Ack
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ack.Type != wsrunner.TypeAck || !ack.OK {
		t.Fatalf("ack: %+v", ack)
	}

	task, err := st.GetTask(ctx, taskMsg.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusSucceeded {
		t.Fatalf("status: %s", task.Status)
	}
}

func TestRunnerStream_WakeAfterHello(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "fleet.db")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	hub := NewWaitHub()
	srv := &Server{
		Store:      st,
		Webhook:    nil,
		Log:        slog.Default(),
		TaskNotify: hub,
	}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/runners/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(wsrunner.Hello{Type: wsrunner.TypeHello, RunnerID: "r-wake", LeaseSeconds: 60}); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		var taskMsg wsrunner.Task
		if err := conn.ReadJSON(&taskMsg); err != nil {
			t.Error(err)
			return
		}
		if taskMsg.Task == nil {
			t.Error("nil task")
			return
		}
		_ = conn.WriteJSON(wsrunner.Complete{
			Type:     wsrunner.TypeComplete,
			TaskID:   taskMsg.Task.ID,
			RunnerID: "r-wake",
			ExitCode: 0,
			Output:   "",
		})
		var ack wsrunner.Ack
		_ = conn.ReadJSON(&ack)
		close(done)
	}()

	<-time.After(50 * time.Millisecond)

	body := `{"command":["echo","x"],"webhook_url":"https://example.com/hook"}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", resp.StatusCode)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for ws task")
	}
}

func TestRunnerStream_AuthRequired(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fleet.db")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	srv := &Server{
		Store:      st,
		Webhook:    nil,
		Log:        slog.Default(),
		TaskNotify: NewWaitHub(),
	}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "secret-token"}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/v1/runners/stream"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("expected dial failure without auth")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got resp=%v err=%v", resp, err)
	}
}

func TestRunnerStream_UnavailableWithoutHub(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fleet.db")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	srv := &Server{Store: st, Webhook: nil, Log: slog.Default(), TaskNotify: nil}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/runners/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("want 503 got %d", resp.StatusCode)
	}
	var eb struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&eb); err != nil {
		t.Fatal(err)
	}
	if eb.Error == "" {
		t.Fatal("expected error body")
	}
}
