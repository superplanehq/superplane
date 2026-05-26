package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/superplane/runner/shared/models"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestPostgresStoreFleetsAndTasks(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:        "fleet-a",
		Labels:    []string{"prod", "tier-1"},
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got.Labels) != 2 {
		t.Fatalf("get fleet: %#v", got)
	}
	if !store.LabelsSubset(got.Labels, []string{"prod"}) {
		t.Fatalf("labels: %#v", got.Labels)
	}

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:        "fleet-a",
		Labels:    []string{"staging"},
		CreatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	got, err = st.GetFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Labels) != 1 || got.Labels[0] != "staging" {
		t.Fatalf("upsert fleet: %#v", got)
	}

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:        "fleet-b",
		Labels:    []string{"prod", "tier-2"},
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	match, err := st.FindFleetByLabels(ctx, []string{"prod", "tier-2"})
	if err != nil {
		t.Fatal(err)
	}
	if match == nil || match.ID != "fleet-b" {
		t.Fatalf("find by labels: %#v", match)
	}

	list, err := st.ListFleets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list fleets: %d", len(list))
	}

	task := &models.Task{
		ID:         "task-1",
		FleetID:    "fleet-a",
		Command:    []string{"echo", "hi"},
		WebhookURL: "https://caller.example/hook",
		Status:     models.StatusQueued,
		CreatedAt:  now,
	}
	if err := st.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	gotTask, err := st.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if gotTask == nil || gotTask.FleetID != "fleet-a" {
		t.Fatalf("get task: %#v", gotTask)
	}

	claimed, err := st.ClaimTask(ctx, "runner-1", "fleet-a", 60*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != "task-1" {
		t.Fatalf("claim: %#v", claimed)
	}

	if err := st.DeleteFleet(ctx, "fleet-b"); err != nil {
		t.Fatal(err)
	}
	list, err = st.ListFleets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "fleet-a" {
		t.Fatalf("after delete fleet-b: %#v", list)
	}
}

func TestPostgresStoreFindFleetByLabelsEmpty(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	match, err := st.FindFleetByLabels(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if match != nil {
		t.Fatalf("expected nil, got %#v", match)
	}
}

func TestPostgresStoreCountTasksByFleet(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	create := func(id, fleetID string, status models.TaskStatus) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    fleetID,
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("a-queued-1", "fleet-a", models.StatusQueued)
	create("a-queued-2", "fleet-a", models.StatusQueued)
	create("a-claimed-1", "fleet-a", models.StatusClaimed)
	create("a-done-1", "fleet-a", models.StatusSucceeded)
	create("a-failed-1", "fleet-a", models.StatusFailed)
	create("a-canceled-1", "fleet-a", models.StatusCanceled)

	create("b-queued-1", "fleet-b", models.StatusQueued)
	create("b-claimed-1", "fleet-b", models.StatusClaimed)
	create("b-claimed-2", "fleet-b", models.StatusClaimed)
	create("b-claimed-3", "fleet-b", models.StatusClaimed)

	qa, ca, err := st.CountTasksByFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if qa != 2 || ca != 1 {
		t.Fatalf("fleet-a counts: got queued=%d claimed=%d, want 2/1", qa, ca)
	}

	qb, cb, err := st.CountTasksByFleet(ctx, "fleet-b")
	if err != nil {
		t.Fatal(err)
	}
	if qb != 1 || cb != 3 {
		t.Fatalf("fleet-b counts: got queued=%d claimed=%d, want 1/3", qb, cb)
	}

	qc, cc, err := st.CountTasksByFleet(ctx, "fleet-missing")
	if err != nil {
		t.Fatal(err)
	}
	if qc != 0 || cc != 0 {
		t.Fatalf("fleet-missing counts: got queued=%d claimed=%d, want 0/0", qc, cc)
	}

	if _, _, err := st.CountTasksByFleet(ctx, ""); err == nil {
		t.Fatalf("expected error for empty fleet id")
	}
}

func TestPostgresStoreListActiveTasks(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	create := func(id string, status models.TaskStatus) {
		t.Helper()
		if err := st.CreateTask(ctx, &models.Task{
			ID:         id,
			FleetID:    "fleet-a",
			Command:    []string{"echo"},
			WebhookURL: "https://example.com/hook",
			Status:     status,
			CreatedAt:  now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	create("queued-1", models.StatusQueued)
	create("claimed-1", models.StatusClaimed)
	create("done-1", models.StatusSucceeded)
	create("failed-1", models.StatusFailed)
	create("canceled-1", models.StatusCanceled)

	active, err := st.ListActiveTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("active tasks: got %d want 2: %#v", len(active), active)
	}
	if active[0].ID != "queued-1" || active[1].ID != "claimed-1" {
		t.Fatalf("order/ids: %#v", active)
	}
}
