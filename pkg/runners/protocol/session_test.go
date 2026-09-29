package protocol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestSessionDeliversTaskCancelAndAcknowledgesCompletion(t *testing.T) {
	var connections atomic.Int32
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/runner/v1/connect" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer access-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer socket.Close()
		connection := connections.Add(1)

		var hello helloMessage
		if err := socket.ReadJSON(&hello); err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		if hello.RunnerID != "runner-1" ||
			hello.FleetID != "linux-amd64" ||
			hello.Version != "0.1.0" {
			t.Errorf("hello = %#v", hello)
			return
		}

		if connection == 1 {
			if err := socket.WriteJSON(taskMessage{
				Type: messageTypeTask,
				Task: json.RawMessage(`{"id":"task-1","execution_mode":"host"}`),
			}); err != nil {
				t.Errorf("write task: %v", err)
				return
			}
			if err := socket.WriteJSON(taskControlMessage{
				Type:   messageTypeCancel,
				TaskID: "task-1",
			}); err != nil {
				t.Errorf("write cancel: %v", err)
				return
			}
			_ = socket.Close()
			return
		}

		if hello.CurrentTaskID != "task-1" {
			t.Errorf("reconnect current task = %q", hello.CurrentTaskID)
			return
		}
		for {
			var raw json.RawMessage
			if err := socket.ReadJSON(&raw); err != nil {
				return
			}
			var envelope messageEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Errorf("decode message: %v", err)
				return
			}
			if envelope.Type != messageTypeComplete {
				continue
			}
			var complete CompleteMessage
			if err := json.Unmarshal(raw, &complete); err != nil {
				t.Errorf("decode completion: %v", err)
				return
			}
			if err := socket.WriteJSON(ackMessage{
				Type:      messageTypeAck,
				RequestID: complete.RequestID,
			}); err != nil {
				t.Errorf("write ack: %v", err)
			}
			return
		}
	}))
	defer server.Close()

	session, err := NewSession(SessionConfig{
		BaseURL:          server.URL,
		Registration:     Registration{RunnerID: "runner-1", FleetID: "linux-amd64", AccessToken: "access-token"},
		Version:          "0.1.0",
		ReconnectMin:     time.Millisecond,
		ReconnectMax:     2 * time.Millisecond,
		ReadTimeout:      time.Second,
		WriteTimeout:     time.Second,
		HandshakeTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	runDone := make(chan error, 1)
	go func() { runDone <- session.Run(t.Context()) }()

	select {
	case raw := <-session.Tasks():
		if !strings.Contains(string(raw), `"id":"task-1"`) {
			t.Fatalf("task = %s", raw)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task")
	}
	select {
	case taskID := <-session.Cancellations():
		if taskID != "task-1" {
			t.Fatalf("cancel task = %q", taskID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cancellation")
	}

	requestID := uuid.NewString()
	if err := session.Complete(t.Context(), CompleteMessage{
		RequestID: requestID,
		TaskID:    "task-1",
		ExitCode:  0,
	}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if connections.Load() < 2 {
		t.Fatalf("connections = %d, want reconnect", connections.Load())
	}
	session.mu.Lock()
	currentTask := session.currentTask
	session.mu.Unlock()
	if currentTask != "" {
		t.Fatalf("current task after acknowledgement = %q", currentTask)
	}
}

func TestConnectURLUsesRunnerV1WebSocketPath(t *testing.T) {
	for _, test := range []struct {
		base string
		want string
	}{
		{base: "http://example.test/", want: "ws://example.test/runner/v1/connect"},
		{base: "https://example.test", want: "wss://example.test/runner/v1/connect"},
	} {
		got, err := connectURL(test.base)
		if err != nil {
			t.Fatalf("connectURL(%q): %v", test.base, err)
		}
		if got != test.want {
			t.Fatalf("connectURL(%q) = %q, want %q", test.base, got, test.want)
		}
	}
}

func TestSessionResendsSameCompletionAfterLostAcknowledgement(t *testing.T) {
	var connections atomic.Int32
	requestIDs := make(chan string, 2)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer socket.Close()
		connection := connections.Add(1)

		var hello helloMessage
		if err := socket.ReadJSON(&hello); err != nil {
			return
		}
		if connection == 1 {
			if err := socket.WriteJSON(taskMessage{
				Type: messageTypeTask,
				Task: json.RawMessage(`{"id":"task-2"}`),
			}); err != nil {
				t.Errorf("write task: %v", err)
				return
			}
		}

		for {
			var raw json.RawMessage
			if err := socket.ReadJSON(&raw); err != nil {
				return
			}
			var completion CompleteMessage
			if err := json.Unmarshal(raw, &completion); err != nil ||
				completion.Type != messageTypeComplete {
				continue
			}
			requestIDs <- completion.RequestID
			if connection == 1 {
				return
			}
			_ = socket.WriteJSON(ackMessage{
				Type:      messageTypeAck,
				RequestID: completion.RequestID,
			})
			return
		}
	}))
	defer server.Close()

	session, err := NewSession(SessionConfig{
		BaseURL:      server.URL,
		Registration: Registration{RunnerID: "runner-2", FleetID: "linux-amd64", AccessToken: "access-token"},
		Version:      "0.1.0",
		ReconnectMin: time.Millisecond,
		ReconnectMax: 2 * time.Millisecond,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = session.Run(ctx) }()
	select {
	case <-session.Tasks():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task")
	}

	const requestID = "5bbf21c6-08cc-4fa6-bd84-421bd6422b68"
	if err := session.Complete(t.Context(), CompleteMessage{
		RequestID: requestID,
		TaskID:    "task-2",
	}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	first := <-requestIDs
	second := <-requestIDs
	if first != requestID || second != requestID {
		t.Fatalf("request IDs = %q, %q", first, second)
	}
}
