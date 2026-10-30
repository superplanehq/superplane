package runner

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/core"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateTaskBuildsRunnerPayload(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "unused")
	tasks := &contexts.RunnerTaskContext{}

	id, err := CreateTask(tasks, CreateTaskParams{
		MachineType:   "e1-large-amd64",
		RunMode:       RunModeBash,
		Script:        "echo hello",
		ExecutionMode: ExecutionModeHost,
		Environment: []TaskEnvironmentVariable{
			{Name: "EXAMPLE", Value: "value"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tasks.CreatedID != id || tasks.FleetID != "e1-large-amd64" {
		t.Fatalf(
			"created id/fleet = %q/%q",
			tasks.CreatedID,
			tasks.FleetID,
		)
	}
	var payload runnerapi.TaskPayload
	if err := json.Unmarshal(tasks.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID != id || payload.Script != "echo hello" ||
		payload.ExecutionMode != ExecutionModeHost {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.LogUploadPolicy != nil {
		t.Fatalf("component payload must not select a log upload policy: %#v", payload.LogUploadPolicy)
	}
}

func TestMissingExecutionBackendDefaultsToLegacy(t *testing.T) {
	if backend := TaskBackendFromExecutionMetadata(nil); backend !=
		core.RunnerTaskBackendLegacy {
		t.Fatalf("metadata backend = %q", backend)
	}
}

func TestCreateTaskRequiresRunnerContext(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "unused")
	if _, err := CreateTask(nil, CreateTaskParams{}); err == nil {
		t.Fatal("must not fall back to the legacy broker")
	}
}

func TestCreateTaskUsesReservedTaskID(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}
	reservedID := uuid.NewString()
	id, err := CreateTask(tasks, CreateTaskParams{TaskID: reservedID, MachineType: "e1-large-amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if id != reservedID || tasks.CreatedID != reservedID {
		t.Fatalf("task ID = %q, created ID = %q, want reserved ID %q", id, tasks.CreatedID, reservedID)
	}
	var payload runnerapi.TaskPayload
	if err := json.Unmarshal(tasks.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID != reservedID {
		t.Fatalf("payload ID = %q, want %q", payload.ID, reservedID)
	}
}

func TestCreateTaskRejectsInvalidReservedTaskID(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}
	if _, err := CreateTask(tasks, CreateTaskParams{TaskID: "invalid", MachineType: "e1-large-amd64"}); err == nil {
		t.Fatal("must reject invalid reserved task ID")
	}
	if tasks.CreatedID != "" {
		t.Fatal("must not create task with invalid reserved ID")
	}
}
