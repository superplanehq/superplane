package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	brokermodels "github.com/superplanehq/superplane/pkg/taskbroker/models"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

func TestListTasksReturnsNonTerminalOnly(t *testing.T) {
	st := openStore(t)

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

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/tasks", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var out api.ListTasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
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
	st := openStore(t)

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/tasks", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var out api.ListTasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Tasks == nil {
		t.Fatal("expected non-nil tasks slice")
	}
	if len(out.Tasks) != 0 {
		t.Fatalf("tasks: %#v", out.Tasks)
	}
}
