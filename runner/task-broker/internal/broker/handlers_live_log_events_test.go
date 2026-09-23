package broker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestLocalLiveLogsIngestAndStream(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	fleetID := "fleet-local-logs"
	runnerID := "runner-local-logs"
	access := mintRunnerAccessToken(t, ctx, st, fleetID, runnerID)

	taskID := uuid.NewString()
	if err := st.CreateTask(ctx, &models.Task{
		ID:         taskID,
		FleetID:    fleetID,
		Status:     models.StatusQueued,
		CreatedAt:  time.Now().UTC(),
		WebhookURL: "https://example.com/hook",
		Command:    []string{"echo", "hi"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimTask(ctx, runnerID, fleetID, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	post, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/"+taskID+"/live-log-events", bytes.NewReader([]byte("hello from runner\n")))
	if err != nil {
		t.Fatal(err)
	}
	post.Header.Set("Authorization", "Bearer "+access)
	resp, err := ts.Client().Do(post)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ingest status: %d", resp.StatusCode)
	}
	srv.closeLiveLogs(taskID)

	get, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/tasks/"+taskID+"/live-logs", nil)
	if err != nil {
		t.Fatal(err)
	}
	get.Header.Set("Authorization", "Bearer tok")
	getResp, err := ts.Client().Do(get)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		t.Fatalf("stream status: %d body: %s", getResp.StatusCode, body)
	}

	scanner := bufio.NewScanner(getResp.Body)
	if !scanner.Scan() {
		t.Fatal("expected one NDJSON line")
	}
	var rec map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	if rec["type"] != "line" || rec["text"] != "hello from runner" {
		t.Fatalf("record: %#v", rec)
	}
}
