package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store"
)

func TestCreateBrokerTaskForwardsEnvironment(t *testing.T) {
	received := make(chan api.CreateTaskRequest, 1)
	fleet := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/tasks" {
			http.NotFound(w, r)
			return
		}
		defer r.Body.Close()
		var req api.CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		received <- req
		writeJSON(w, http.StatusCreated, api.CreateTaskResponse{ID: "fleet-task-1"})
	}))
	defer fleet.Close()

	st, err := store.OpenSQLite(filepath.Join(t.TempDir(), "broker.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:        "fleet-1",
		BaseURL:   fleet.URL,
		Labels:    []string{"e2e"},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{
		Store:     st,
		PublicURL: "http://broker.example",
		Log:       slog.Default(),
		HTTP:      fleet.Client(),
	}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body, err := json.Marshal(api.BrokerCreateTaskRequest{
		CreateTaskRequest: api.CreateTaskRequest{
			Commands:   []string{"echo \"$COMMIT_AUTHOR\""},
			WebhookURL: "https://example.com/hook",
			Environment: []api.EnvironmentVariable{
				{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
				{Name: "SPECIAL", Value: "line one\nline two=ok"},
			},
		},
		FleetID: "fleet-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create broker task: %d %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	select {
	case upstream := <-received:
		if len(upstream.Environment) != 2 {
			t.Fatalf("environment: %#v", upstream.Environment)
		}
		if upstream.Environment[0].Name != "COMMIT_AUTHOR" || upstream.Environment[0].Value != "alice@example.com" {
			t.Fatalf("first env: %#v", upstream.Environment[0])
		}
		if upstream.Environment[1].Name != "SPECIAL" || upstream.Environment[1].Value != "line one\nline two=ok" {
			t.Fatalf("second env: %#v", upstream.Environment[1])
		}
		if !strings.Contains(upstream.WebhookURL, "/v1/webhooks/complete/") {
			t.Fatalf("upstream webhook should be broker relay, got %q", upstream.WebhookURL)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for upstream create")
	}
}
