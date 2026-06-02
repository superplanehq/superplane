package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
)

func TestRunHTTP_exitAfterEachTaskWhenCompleteFails(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh not on PATH: %v", err)
	}

	var completeCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks/claim":
			writeJSON(w, http.StatusOK, api.ClaimTaskResponse{
				Task: &api.TaskPayload{
					ID:      "task-one-shot",
					Command: []string{sh, "-c", "true"},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks/task-one-shot/complete":
			completeCalls++
			http.Error(w, "broker unavailable", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	a := &Agent{Config: Config{
		BaseURL:           srv.URL,
		FleetID:           "test-fleet",
		RunnerID:          "i-test",
		Transport:         "http",
		ExitAfterEachTask: true,
		TaskWorkDir:       t.TempDir(),
	}}
	if err := a.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if completeCalls != 1 {
		t.Fatalf("complete calls = %d, want 1", completeCalls)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
