package store_test

import (
	"context"
	"testing"
	"time"

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
		BaseURL:   "http://fleet-a.example",
		AuthToken: "secret",
		Labels:    []string{"prod", "tier-1"},
		CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.BaseURL != "http://fleet-a.example" || got.AuthToken != "secret" {
		t.Fatalf("get fleet: %#v", got)
	}
	if !store.LabelsSubset(got.Labels, []string{"prod"}) {
		t.Fatalf("labels: %#v", got.Labels)
	}

	// Upsert replaces fields.
	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:        "fleet-a",
		BaseURL:   "http://fleet-a-new.example",
		Labels:    []string{"staging"},
		CreatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	got, err = st.GetFleet(ctx, "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != "http://fleet-a-new.example" || got.AuthToken != "" {
		t.Fatalf("upsert fleet: %#v", got)
	}

	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:        "fleet-b",
		BaseURL:   "http://fleet-b.example",
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

	taskID := "broker-task-1"
	if err := st.InsertBrokerTask(ctx, &brokermodels.BrokerTask{
		ID:               taskID,
		FleetID:          "fleet-a",
		CallerWebhookURL: "https://caller.example/hook",
		CreatedAt:        now,
	}); err != nil {
		t.Fatal(err)
	}

	task, err := st.GetBrokerTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task == nil || task.FleetTaskID != "" {
		t.Fatalf("get task: %#v", task)
	}

	if err := st.UpdateBrokerTaskFleetTaskID(ctx, taskID, "fleet-task-99"); err != nil {
		t.Fatal(err)
	}
	task, err = st.GetBrokerTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.FleetTaskID != "fleet-task-99" {
		t.Fatalf("fleet task id: %q", task.FleetTaskID)
	}

	if err := st.DeleteBrokerTask(ctx, taskID); err != nil {
		t.Fatal(err)
	}
	task, err = st.GetBrokerTask(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if task != nil {
		t.Fatalf("expected nil after delete, got %#v", task)
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

func TestPostgresStoreUpdateBrokerTaskMissing(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()

	err := st.UpdateBrokerTaskFleetTaskID(context.Background(), "missing", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}
