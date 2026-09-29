package runner

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
)

type fakeRunnerTaskContext struct {
	enabled        bool
	createdID      string
	createdFleet   string
	createdPayload []byte
	task           *core.RunnerTask
	canceledID     string
}

func (f *fakeRunnerTaskContext) IntegratedBackendEnabled() (bool, error) {
	return f.enabled, nil
}

func (f *fakeRunnerTaskContext) Create(id, fleetID string, payload []byte) error {
	f.createdID = id
	f.createdFleet = fleetID
	f.createdPayload = append([]byte(nil), payload...)
	return nil
}

func (f *fakeRunnerTaskContext) Find(string) (*core.RunnerTask, error) {
	return f.task, nil
}

func (f *fakeRunnerTaskContext) RequestCancel(id string) error {
	f.canceledID = id
	return nil
}

func TestIntegratedTaskClientCreatesTypedPayload(t *testing.T) {
	tasks := &fakeRunnerTaskContext{enabled: true}
	client, backend, err := NewTaskClient(nil, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if backend != core.RunnerTaskBackendIntegrated {
		t.Fatalf("backend = %q", backend)
	}

	id, err := client.CreateTask(CreateTaskParams{
		MachineType:   "e1-large-amd64",
		RunMode:       RunModeBash,
		Script:        "echo hello",
		ExecutionMode: ExecutionModeHost,
		Environment: []BrokerEnvironmentVariable{
			{Name: "EXAMPLE", Value: "value"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tasks.createdID != id || tasks.createdFleet != "e1-large-amd64" {
		t.Fatalf(
			"created id/fleet = %q/%q",
			tasks.createdID,
			tasks.createdFleet,
		)
	}
	var payload runnerapi.TaskPayload
	if err := json.Unmarshal(tasks.createdPayload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID != id || payload.Script != "echo hello" ||
		payload.ExecutionMode != ExecutionModeHost {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestIntegratedTaskClientMapsTerminalTaskAndCancels(t *testing.T) {
	finished := time.Now()
	exitCode := 1
	tasks := &fakeRunnerTaskContext{
		task: &core.RunnerTask{
			ID:           "task-1",
			State:        "failed",
			Result:       json.RawMessage(`{"reason":"test"}`),
			ExitCode:     &exitCode,
			ErrorMessage: "failed",
			FinishedAt:   &finished,
		},
	}
	client := &integratedTaskClient{tasks: tasks}
	task, err := client.FetchTaskStatus("task-1")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "failed" || task.effectiveExitCode() != 1 ||
		task.Error != "failed" {
		t.Fatalf("task = %#v", task)
	}
	if err := client.CancelTask("task-1"); err != nil {
		t.Fatal(err)
	}
	if tasks.canceledID != "task-1" {
		t.Fatalf("canceled ID = %q", tasks.canceledID)
	}
}

func TestMissingExecutionBackendDefaultsToLegacy(t *testing.T) {
	if backend := TaskBackendFromExecutionMetadata(nil); backend !=
		core.RunnerTaskBackendLegacy {
		t.Fatalf("metadata backend = %q", backend)
	}
	if backend := taskBackendFromState(nil); backend !=
		core.RunnerTaskBackendLegacy {
		t.Fatalf("state backend = %q", backend)
	}
}

var _ core.RunnerTaskContext = (*fakeRunnerTaskContext)(nil)
