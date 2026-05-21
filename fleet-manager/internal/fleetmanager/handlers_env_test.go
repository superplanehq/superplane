package fleetmanager

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

	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
)

func TestCreateClaimTaskRoundTripsEnvironmentAndDoesNotExposeInStatusOrWebhook(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "fleet.db")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	received := make(chan map[string]any, 1)
	wh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("webhook json: %v", err)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		received <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer wh.Close()

	srv := &Server{
		Store:   st,
		Webhook: &webhook.Sender{Client: http.DefaultClient, Retries: 1},
		Log:     slog.Default(),
	}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{}))
	defer ts.Close()

	createBody, err := json.Marshal(api.CreateTaskRequest{
		Command:    []string{"sh", "-c", "printf %s \"$COMMIT_AUTHOR\""},
		WebhookURL: wh.URL,
		Environment: []api.EnvironmentVariable{
			{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
			{Name: "EMPTY_OK", Value: ""},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(ts.URL+"/v1/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create task: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var created api.CreateTaskResponse
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}

	claimBody, err := json.Marshal(api.ClaimTaskRequest{RunnerID: "runner-1", LeaseSeconds: 300})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(ts.URL+"/v1/tasks/claim", "application/json", bytes.NewReader(claimBody))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("claim task: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var claim api.ClaimTaskResponse
	if err := json.Unmarshal(body, &claim); err != nil {
		t.Fatal(err)
	}
	if claim.Task == nil || claim.Task.ID != created.ID {
		t.Fatalf("claim payload: %+v", claim.Task)
	}
	if len(claim.Task.Environment) != 2 || claim.Task.Environment[0].Value != "alice@example.com" || claim.Task.Environment[1].Value != "" {
		t.Fatalf("claim environment: %#v", claim.Task.Environment)
	}

	resp, err = http.Get(ts.URL + "/v1/tasks/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get task: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var status map[string]any
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatal(err)
	}
	if _, ok := status["environment"]; ok {
		t.Fatalf("status must not expose environment: %s", string(body))
	}

	completeBody, err := json.Marshal(api.CompleteTaskRequest{
		RunnerID: "runner-1",
		ExitCode: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(ts.URL+"/v1/tasks/"+created.ID+"/complete", "application/json", bytes.NewReader(completeBody))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("complete task: %d", resp.StatusCode)
	}

	select {
	case payload := <-received:
		if _, ok := payload["environment"]; ok {
			t.Fatalf("webhook must not expose environment: %#v", payload)
		}
		if payload["status"] != string(models.StatusSucceeded) {
			t.Fatalf("webhook status: %#v", payload["status"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for webhook")
	}

	task, err := st.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(task.Environment) != 0 {
		t.Fatalf("completed task should not retain environment: %#v", task.Environment)
	}
}

func TestCreateTaskRejectsInvalidEnvironment(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fleet.db")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	srv := &Server{Store: st, Webhook: nil, Log: slog.Default()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{}))
	defer ts.Close()

	createBody := []byte(`{"command":["true"],"webhook_url":"https://example.com/hook","environment":[{"name":"BAD-NAME","value":"x"}]}`)
	resp, err := http.Post(ts.URL+"/v1/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 400: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
