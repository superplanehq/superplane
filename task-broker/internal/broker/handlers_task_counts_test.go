package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestFleetTaskCountsHandler(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	if err := st.CreateFleet(ctx, &brokermodels.Fleet{ID: "fleet-a", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	create := func(id string, status models.TaskStatus) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID: id, FleetID: "fleet-a", Command: []string{"echo"},
			WebhookURL: "https://example.com/h", Status: status, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	create("q1", models.StatusQueued)
	create("q2", models.StatusQueued)
	create("c1", models.StatusClaimed)
	create("done", models.StatusSucceeded)

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/fleets/fleet-a/task-counts", nil)
		req.Header.Set("Authorization", "Bearer tok")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: %d", resp.StatusCode)
		}
		var got api.FleetTaskCountsResponse
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.Queued != 2 || got.Claimed != 1 {
			t.Fatalf("counts: %#v", got)
		}
	})

	t.Run("missing fleet 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/fleets/nope/task-counts", nil)
		req.Header.Set("Authorization", "Bearer tok")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status: %d", resp.StatusCode)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/fleets/fleet-a/task-counts", nil)
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status: %d", resp.StatusCode)
		}
	})
}
