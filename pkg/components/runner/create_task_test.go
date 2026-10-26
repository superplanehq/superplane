package runner

import (
	"encoding/json"
	"testing"

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
