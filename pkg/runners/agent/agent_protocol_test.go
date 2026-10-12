package agent

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/superplanehq/superplane/pkg/runners/protocol"
)

func TestAgentCompletesOnlyAfterEveryLogChunkIsAcknowledged(t *testing.T) {
	logUploadStarted := make(chan struct{}, 1)
	releaseLogUpload := make(chan struct{})
	completionReceived := make(chan protocol.CompleteMessage, 1)
	upgrader := websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /runner/v1/connect", func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer socket.Close()

		var hello map[string]any
		if err := socket.ReadJSON(&hello); err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		if err := socket.WriteJSON(map[string]any{
			"type": "task",
			"task": map[string]any{
				"id":             "task-1",
				"run_mode":       "argv",
				"command":        []string{"sh", "-c", "printf 'hello from runner\\n'"},
				"execution_mode": "host",
			},
		}); err != nil {
			t.Errorf("write task: %v", err)
			return
		}

		for {
			var raw json.RawMessage
			if err := socket.ReadJSON(&raw); err != nil {
				return
			}
			var envelope struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Errorf("decode message: %v", err)
				return
			}
			if envelope.Type != "complete" {
				continue
			}
			var completion protocol.CompleteMessage
			if err := json.Unmarshal(raw, &completion); err != nil {
				t.Errorf("decode completion: %v", err)
				return
			}
			completionReceived <- completion
			_ = socket.WriteJSON(map[string]any{
				"type":       "ack",
				"request_id": completion.RequestID,
			})
			_ = socket.WriteJSON(map[string]any{"type": "shutdown"})
			return
		}
	})
	mux.HandleFunc(
		"PUT /runner/v1/tasks/{task_id}/logs/chunks/{sequence}",
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			select {
			case logUploadStarted <- struct{}{}:
			default:
			}
			<-releaseLogUpload
			w.WriteHeader(http.StatusNoContent)
		},
	)
	server := httptest.NewServer(mux)
	defer server.Close()

	runner := &Agent{
		Config: Config{
			BaseURL: server.URL,
			Registration: protocol.Registration{
				RunnerID:    "runner-1",
				FleetID:     "linux-amd64",
				AccessToken: "access-token",
				Ephemeral:   true,
			},
			Version:           "0.1.0",
			TaskWorkDir:       t.TempDir(),
			LogSpoolDirectory: t.TempDir(),
			LogChunkBytes:     1024,
			LogSpoolMaxBytes:  4096,
			ReconnectMin:      time.Millisecond,
			ReconnectMax:      2 * time.Millisecond,
			Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		HTTP: server.Client(),
	}

	runDone := make(chan error, 1)
	go func() { runDone <- runner.Run(t.Context()) }()

	select {
	case <-logUploadStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for log upload")
	}
	select {
	case completion := <-completionReceived:
		t.Fatalf("completion arrived before log acknowledgement: %#v", completion)
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseLogUpload)

	select {
	case completion := <-completionReceived:
		if completion.TaskID != "task-1" || completion.ExitCode != 0 {
			t.Fatalf("completion = %#v", completion)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for completion")
	}
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent to stop")
	}
}

func TestAgentRequestsShutdownBeforeCancelingCurrentTask(t *testing.T) {
	taskSent := make(chan struct{})
	shutdownReceived := make(chan map[string]any, 1)
	completionReceived := make(chan protocol.CompleteMessage, 1)
	upgrader := websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /runner/v1/connect", func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer socket.Close()

		var hello map[string]any
		if err := socket.ReadJSON(&hello); err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		if err := socket.WriteJSON(map[string]any{
			"type": "task",
			"task": map[string]any{
				"id":             "task-1",
				"run_mode":       "argv",
				"command":        []string{"sleep", "30"},
				"execution_mode": "host",
			},
		}); err != nil {
			t.Errorf("write task: %v", err)
			return
		}
		close(taskSent)

		for {
			var raw json.RawMessage
			if err := socket.ReadJSON(&raw); err != nil {
				return
			}
			var envelope struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Errorf("decode message: %v", err)
				return
			}
			switch envelope.Type {
			case "shutdown_request":
				var request map[string]any
				if err := json.Unmarshal(raw, &request); err != nil {
					t.Errorf("decode shutdown request: %v", err)
					return
				}
				shutdownReceived <- request
				if err := socket.WriteJSON(map[string]any{
					"type":       "cancel",
					"task_id":    "task-1",
					"request_id": request["request_id"],
					"reason":     "signal",
				}); err != nil {
					t.Errorf("write cancel: %v", err)
					return
				}
			case "complete":
				var completion protocol.CompleteMessage
				if err := json.Unmarshal(raw, &completion); err != nil {
					t.Errorf("decode completion: %v", err)
					return
				}
				completionReceived <- completion
				if err := socket.WriteJSON(map[string]any{
					"type":       "ack",
					"request_id": completion.RequestID,
				}); err != nil {
					t.Errorf("write acknowledgement: %v", err)
					return
				}
				_ = socket.WriteJSON(map[string]any{"type": "shutdown"})
				return
			}
		}
	})
	mux.HandleFunc(
		"PUT /runner/v1/tasks/{task_id}/logs/chunks/{sequence}",
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		},
	)
	server := httptest.NewServer(mux)
	defer server.Close()

	shutdown := make(chan struct{}, 2)
	var agentLogs bytes.Buffer
	runner := &Agent{
		Config: Config{
			BaseURL: server.URL,
			Registration: protocol.Registration{
				RunnerID:    "runner-1",
				FleetID:     "linux-amd64",
				AccessToken: "access-token",
				Ephemeral:   true,
			},
			Version:           "0.1.0",
			TaskWorkDir:       t.TempDir(),
			LogSpoolDirectory: t.TempDir(),
			LogChunkBytes:     1024,
			LogSpoolMaxBytes:  4096,
			ReconnectMin:      time.Millisecond,
			ReconnectMax:      2 * time.Millisecond,
			Shutdown:          shutdown,
			ShutdownGrace:     2 * time.Second,
			Log:               slog.New(slog.NewTextHandler(&agentLogs, nil)),
		},
		HTTP: server.Client(),
	}

	runDone := make(chan error, 1)
	go func() { runDone <- runner.Run(t.Context()) }()
	select {
	case <-taskSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task delivery")
	}
	time.Sleep(50 * time.Millisecond)
	shutdown <- struct{}{}

	select {
	case request := <-shutdownReceived:
		if request["current_task_id"] != "task-1" || request["reason"] != "signal" {
			t.Fatalf("shutdown request = %#v", request)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown request")
	}
	select {
	case completion := <-completionReceived:
		if !completion.Canceled || completion.TaskID != "task-1" {
			t.Fatalf("completion = %#v", completion)
		}
	case <-time.After(2 * time.Second):
		select {
		case err := <-runDone:
			t.Fatalf("runner stopped before completion: %v", err)
		default:
		}
		t.Fatal("timed out waiting for canceled completion")
	}
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for runner shutdown")
	}
	if !bytes.Contains(agentLogs.Bytes(), []byte("runner requesting controlled shutdown")) {
		t.Fatalf("missing shutdown request log:\n%s", agentLogs.String())
	}
	if !bytes.Contains(agentLogs.Bytes(), []byte("runner received shutdown instruction")) {
		t.Fatalf("missing shutdown instruction log:\n%s", agentLogs.String())
	}
}

func TestReusableAgentExecutesMoreThanOneTask(t *testing.T) {
	upgrader := websocket.Upgrader{}
	completions := make(chan string, 2)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /runner/v1/connect", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runner/v1/connect" {
			http.NotFound(w, r)
			return
		}
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer socket.Close()

		var hello map[string]any
		if err := socket.ReadJSON(&hello); err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		for _, taskID := range []string{"task-1", "task-2"} {
			if err := socket.WriteJSON(map[string]any{
				"type": "task",
				"task": map[string]any{
					"id":             taskID,
					"run_mode":       "argv",
					"command":        []string{"sh", "-c", "true"},
					"execution_mode": "host",
				},
			}); err != nil {
				t.Errorf("write task: %v", err)
				return
			}
			for {
				var message protocol.CompleteMessage
				if err := socket.ReadJSON(&message); err != nil {
					t.Errorf("read completion: %v", err)
					return
				}
				if message.Type != "complete" {
					continue
				}
				completions <- message.TaskID
				if err := socket.WriteJSON(map[string]any{
					"type":       "ack",
					"request_id": message.RequestID,
				}); err != nil {
					t.Errorf("write acknowledgement: %v", err)
					return
				}
				break
			}
		}
		_ = socket.WriteJSON(map[string]any{"type": "shutdown"})
	})
	mux.HandleFunc(
		"PUT /runner/v1/tasks/{task_id}/logs/chunks/{sequence}",
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		},
	)
	server := httptest.NewServer(mux)
	defer server.Close()

	runner := &Agent{
		Config: Config{
			BaseURL: server.URL,
			Registration: protocol.Registration{
				RunnerID:    "runner-1",
				FleetID:     "linux-amd64",
				AccessToken: "access-token",
				Ephemeral:   false,
			},
			Version:           "0.1.0",
			TaskWorkDir:       t.TempDir(),
			LogSpoolDirectory: t.TempDir(),
			LogChunkBytes:     1024,
			LogSpoolMaxBytes:  4096,
			ReconnectMin:      time.Millisecond,
			ReconnectMax:      2 * time.Millisecond,
			Log:               slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
		HTTP: server.Client(),
	}

	if err := runner.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, expected := range []string{"task-1", "task-2"} {
		select {
		case actual := <-completions:
			if actual != expected {
				t.Fatalf("completion task = %q, want %q", actual, expected)
			}
		default:
			t.Fatalf("missing completion for %s", expected)
		}
	}
}
