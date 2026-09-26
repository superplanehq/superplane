package broker

import (
	"bytes"
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

func TestDrainRunnersTreatsPersistedClaimedTaskAsBusy(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	if err := st.CreateFleet(ctx, &brokermodels.Fleet{ID: "fleet-a", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTask(ctx, &models.Task{
		ID:         "task-claimed",
		FleetID:    "fleet-a",
		Command:    []string{"echo"},
		WebhookURL: "https://example.com/hook",
		Status:     models.StatusClaimed,
		CreatedAt:  now,
		RunnerID:   "i-claimed",
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st, RunnerDrain: NewRunnerDrainHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	body, err := json.Marshal(api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"i-claimed", "i-idle"},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/runners/drain", bytes.NewReader(body))
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

	var got api.DrainRunnersResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	statuses := drainStatusesByRunner(got.Runners)
	if statuses["i-claimed"].State != api.DrainRunnerStateBusy {
		t.Fatalf("claimed runner status: %#v", statuses["i-claimed"])
	}
	if statuses["i-claimed"].ActiveTaskID != "task-claimed" {
		t.Fatalf("active task id: %q", statuses["i-claimed"].ActiveTaskID)
	}
	if statuses["i-idle"].State != api.DrainRunnerStateDrained {
		t.Fatalf("idle runner status: %#v", statuses["i-idle"])
	}
}

func drainStatusesByRunner(statuses []api.DrainRunnerStatus) map[string]api.DrainRunnerStatus {
	out := make(map[string]api.DrainRunnerStatus, len(statuses))
	for _, status := range statuses {
		out[status.RunnerID] = status
	}
	return out
}
