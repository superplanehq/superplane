package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/wsrunner"
)

func TestRunWebSocketFallsBackToHTTPWhenCompletionAckIsLost(t *testing.T) {
	completeCalls := runWebSocketCompletionFallbackTest(t, http.StatusNoContent)
	if completeCalls != 1 {
		t.Fatalf("HTTP fallback complete calls = %d, want 1", completeCalls)
	}
}

func TestRunWebSocketReturnsErrorWhenCompletionFallbackFails(t *testing.T) {
	var completeCalls int32
	err := runWebSocketCompletionFallbackServer(t, http.StatusInternalServerError, &completeCalls).Run(testAgentContext(t))
	if err == nil {
		t.Fatal("Run: expected completion delivery error")
	}
	if !strings.Contains(err.Error(), "http fallback failed") {
		t.Fatalf("Run error = %v, want HTTP fallback failure", err)
	}
	if atomic.LoadInt32(&completeCalls) != 1 {
		t.Fatalf("HTTP fallback complete calls = %d, want 1", completeCalls)
	}
}

func runWebSocketCompletionFallbackTest(t *testing.T, fallbackStatus int) int32 {
	t.Helper()
	var completeCalls int32
	a := runWebSocketCompletionFallbackServer(t, fallbackStatus, &completeCalls)
	if err := a.Run(testAgentContext(t)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return atomic.LoadInt32(&completeCalls)
}

func runWebSocketCompletionFallbackServer(t *testing.T, fallbackStatus int, completeCalls *int32) *Agent {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not on PATH: %v", err)
	}

	upgrader := websocket.Upgrader{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/runners/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		var hello wsrunner.Hello
		if err := conn.ReadJSON(&hello); err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		if hello.Type != wsrunner.TypeHello {
			t.Errorf("hello type = %q, want %q", hello.Type, wsrunner.TypeHello)
			return
		}
		if err := conn.WriteJSON(wsrunner.Task{
			Type: wsrunner.TypeTask,
			Task: &api.TaskPayload{
				ID:      "task-ws-fallback",
				Command: []string{sh, "-c", "true"},
			},
		}); err != nil {
			t.Errorf("write task: %v", err)
			return
		}
		var complete wsrunner.Complete
		if err := conn.ReadJSON(&complete); err != nil {
			t.Errorf("read complete: %v", err)
			return
		}
		if complete.Type != wsrunner.TypeComplete || complete.TaskID != "task-ws-fallback" {
			t.Errorf("complete = %#v", complete)
		}
	})
	mux.HandleFunc("/v1/tasks/task-ws-fallback/complete", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(completeCalls, 1)
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(fallbackStatus)
		if fallbackStatus != http.StatusNoContent {
			_, _ = fmt.Fprint(w, "broker unavailable")
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &Agent{Config: Config{
		BaseURL:              srv.URL,
		FleetID:              "test-fleet",
		RunnerID:             "i-test",
		ExitAfterEachTask:    true,
		TaskWorkDir:          t.TempDir(),
		CompleteRetryBackoff: []time.Duration{0},
	}}
}

func testAgentContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return ctx
}
