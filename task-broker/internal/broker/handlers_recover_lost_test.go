package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestRecoverLostRunnersRequeuesClaimedTask(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	taskID := uuid.NewString()
	if err := st.CreateTask(ctx, &models.Task{
		ID:         taskID,
		FleetID:    "fleet-a",
		Status:     models.StatusQueued,
		CreatedAt:  time.Now().UTC(),
		WebhookURL: "https://example.com/hook",
		Command:    []string{"echo", "hi"},
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := st.ClaimTask(ctx, "runner-1", "fleet-a", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil {
		t.Fatal("expected task to be claimed")
	}

	drain := NewRunnerDrainHub()
	if !drain.TryStartClaim("runner-1") {
		t.Fatal("expected claim to start")
	}
	drain.FinishClaim("runner-1", taskID)
	srv := &Server{Store: st, TaskNotify: NewWaitHub(), RunnerDrain: drain}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	body, err := json.Marshal(api.RecoverLostRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"runner-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/runners/recover-lost", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	var got api.RecoverLostRunnersResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("tasks: %#v", got.Tasks)
	}
	if got.Tasks[0].RunnerID != "runner-1" ||
		got.Tasks[0].TaskID != taskID ||
		got.Tasks[0].State != api.RunnerTaskRecoveryStateRequeued {
		t.Fatalf("task recovery: %#v", got.Tasks[0])
	}

	task, err := st.GetTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != models.StatusQueued || task.RunnerID != "" {
		t.Fatalf("task after recovery: %#v", task)
	}

	statuses := drain.Drain("fleet-a", []string{"runner-1"})
	if len(statuses) != 1 || statuses[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("runner drain status after recovery: %#v", statuses)
	}
}
