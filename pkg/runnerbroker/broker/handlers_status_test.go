package broker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/runnerbroker/models"
)

func TestTaskStatusResponseIncludesTimelineAndFleet(t *testing.T) {
	created := time.Date(2026, 5, 24, 20, 1, 0, 0, time.UTC)
	claimed := created.Add(2 * time.Second)
	lease := claimed.Add(10 * time.Minute)
	task := &models.Task{
		ID:            "task-1",
		FleetID:       "e1-tiny-amd64",
		Status:        models.StatusClaimed,
		CreatedAt:     created,
		ClaimedAt:     &claimed,
		LeaseUntil:    &lease,
		RunnerID:      "i-0abc123",
		ExecutionMode: models.ExecutionDocker,
		DockerImage:   "alpine:3.20",
	}

	resp := taskStatusResponse(task, &Server{})
	if resp.FleetID != "e1-tiny-amd64" {
		t.Fatalf("fleet_id: %q", resp.FleetID)
	}
	if resp.RunnerID != "i-0abc123" {
		t.Fatalf("runner_id: %q", resp.RunnerID)
	}
	if !resp.CreatedAt.Equal(created) {
		t.Fatalf("created_at: %v", resp.CreatedAt)
	}
	if resp.ClaimedAt == nil || !resp.ClaimedAt.Equal(claimed) {
		t.Fatalf("claimed_at: %#v", resp.ClaimedAt)
	}
	if resp.LeaseUntil == nil || !resp.LeaseUntil.Equal(lease) {
		t.Fatalf("lease_until: %#v", resp.LeaseUntil)
	}
	if resp.ExecutionMode != "docker" {
		t.Fatalf("execution_mode: %q", resp.ExecutionMode)
	}
	if resp.DockerImage != "alpine:3.20" {
		t.Fatalf("docker_image: %q", resp.DockerImage)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"fleet_id", "created_at", "claimed_at", "lease_until", "runner_id", "execution_mode", "docker_image"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing json field %q in %s", key, string(b))
		}
	}
}
