package broker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func newListTestServer(t *testing.T, st *store.PostgresStore) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(NewRouter(&Server{Store: st}, RouterOptions{AuthToken: "tok"}))
	t.Cleanup(ts.Close)
	return ts
}

func getListTasks(t *testing.T, ts *httptest.Server, query string) (api.ListTasksResponse, int) {
	t.Helper()
	url := ts.URL + "/v1/tasks"
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return api.ListTasksResponse{}, resp.StatusCode
	}
	var out api.ListTasksResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	return out, resp.StatusCode
}

func TestListTasksReturnsNonTerminalOnly(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	if err := st.CreateTask(ctx, &models.Task{
		ID: "active-queued", FleetID: "f1", Command: []string{"echo"},
		WebhookURL: "https://example.com/h", Status: models.StatusQueued, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTask(ctx, &models.Task{
		ID: "active-claimed", FleetID: "f1", Command: []string{"echo"},
		WebhookURL: "https://example.com/h", Status: models.StatusClaimed,
		CreatedAt: now, RunnerID: "runner-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTask(ctx, &models.Task{
		ID: "terminal", FleetID: "f1", Command: []string{"echo"},
		WebhookURL: "https://example.com/h", Status: models.StatusSucceeded, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	_ = st.CreateFleet(ctx, &brokermodels.Fleet{ID: "f1", CreatedAt: now})

	out, status := getListTasks(t, newListTestServer(t, st), "")
	if status != http.StatusOK {
		t.Fatalf("status: %d", status)
	}
	if len(out.Tasks) != 2 {
		t.Fatalf("tasks: %#v", out.Tasks)
	}
	if out.Tasks[0].ID != "active-queued" || out.Tasks[0].Status != "queued" {
		t.Fatalf("first: %#v", out.Tasks[0])
	}
	if out.Tasks[1].ID != "active-claimed" || out.Tasks[1].RunnerID != "runner-1" {
		t.Fatalf("second: %#v", out.Tasks[1])
	}
}

func TestListTasksEmpty(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	out, status := getListTasks(t, newListTestServer(t, st), "")
	if status != http.StatusOK {
		t.Fatalf("status: %d", status)
	}
	if out.Tasks == nil {
		t.Fatal("expected non-nil tasks slice")
	}
	if len(out.Tasks) != 0 {
		t.Fatalf("tasks: %#v", out.Tasks)
	}
}

func TestListTasksByRunnerID(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	labels := map[string]string{models.LabelCanvasID: "canvas-uuid"}
	if err := st.CreateTask(ctx, &models.Task{
		ID: "done-task", FleetID: "f1", Command: []string{"echo"},
		WebhookURL: "https://example.com/h", Status: models.StatusSucceeded,
		CreatedAt: now.Add(time.Second), RunnerID: "i-aaa", Labels: labels,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTask(ctx, &models.Task{
		ID: "claimed-task", FleetID: "f1", Command: []string{"echo"},
		WebhookURL: "https://example.com/h", Status: models.StatusClaimed,
		CreatedAt: now, RunnerID: "i-aaa",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTask(ctx, &models.Task{
		ID: "other-runner", FleetID: "f1", Command: []string{"echo"},
		WebhookURL: "https://example.com/h", Status: models.StatusSucceeded,
		CreatedAt: now.Add(2 * time.Second), RunnerID: "i-bbb", Labels: labels,
	}); err != nil {
		t.Fatal(err)
	}

	ts := newListTestServer(t, st)
	out, status := getListTasks(t, ts, "runner_id=i-aaa")
	if status != http.StatusOK {
		t.Fatalf("status: %d", status)
	}
	if len(out.Tasks) != 2 {
		t.Fatalf("tasks: %#v", out.Tasks)
	}
	if out.Tasks[0].ID != "done-task" || out.Tasks[0].Status != "succeeded" {
		t.Fatalf("first: %#v", out.Tasks[0])
	}
	if out.Tasks[0].Labels[models.LabelCanvasID] != "canvas-uuid" {
		t.Fatalf("labels: %#v", out.Tasks[0].Labels)
	}
	if out.Tasks[1].ID != "claimed-task" {
		t.Fatalf("second: %#v", out.Tasks[1])
	}

	_, status = getListTasks(t, ts, "runner_id=")
	if status != http.StatusBadRequest {
		t.Fatalf("empty runner_id status: %d", status)
	}
}
