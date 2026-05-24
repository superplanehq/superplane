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
