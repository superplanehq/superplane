package runner

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestBillableSeconds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration time.Duration
		expected int64
	}{
		{"sub-second rounds up", 117 * time.Millisecond, 1},
		{"whole second", time.Second, 1},
		{"partial second rounds up", 7739 * time.Millisecond, 8},
		{"exact multiple", 5 * time.Minute, 300},
		{"zero", 0, 0},
		{"sub-second negative from clock skew", -117 * time.Millisecond, 0},
		{"negative", -30 * time.Second, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, billableSeconds(tc.duration))
		})
	}
}

func TestIntegratedRunnerTaskRecordsSelectedFleetUsage(t *testing.T) {
	t.Setenv("TASK_BROKER_FLEET_ID", "local")
	t.Setenv("TASK_BROKER_BASE_URL", "http://localhost:8091")

	started := time.Now()
	finished := started.Add(1500 * time.Millisecond)
	exitCode := 0
	tasks := &contexts.RunnerTaskContext{
		Task: &core.RunnerTask{
			ID:         "task-integrated",
			State:      "succeeded",
			ExitCode:   &exitCode,
			StartedAt:  &started,
			FinishedAt: &finished,
		},
	}
	state := &contexts.ExecutionStateContext{}
	scheduled := &contexts.RequestContext{}
	configuration := map[string]any{"machine_type": "aws-large-amd64"}
	require.NoError(t, afterRunnerTaskCreated(core.ExecutionContext{
		OrganizationID: "organization-1",
		Configuration:  configuration,
		ExecutionState: state,
		Requests:       scheduled,
	}, tasks.Task.ID))
	assert.Equal(t, "aws-large-amd64", state.KVs[executionKVMachineType])
	assert.Equal(t, "aws-large-amd64", state.KVs[executionKVFleetID])

	recorder := &recordingUsage{}
	require.NoError(t, pollTask(core.ActionHookContext{
		Configuration:  configuration,
		Parameters:     scheduled.Params,
		ExecutionState: state,
		Requests:       &contexts.RequestContext{},
		RunnerTasks:    tasks,
		Usage:          recorder,
	}, "runnerRunBash.finished"))

	require.Len(t, recorder.computes, 1)
	assert.Equal(t, "aws-large-amd64", recorder.computes[0].MachineType)
	assert.Equal(t, "aws-large-amd64", recorder.computes[0].FleetID)
	assert.Equal(t, int64(2), recorder.computes[0].DurationSeconds)
	assert.Equal(
		t,
		models.UsageIdempotencyKeyRunner+":compute:task-integrated",
		recorder.computes[0].IdempotencyKey,
	)
}

func TestPollTaskRecordsUsageWhenSuperPlaneAlreadyFinished(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: canceledTaskWithUsage()}

	recorder := &recordingUsage{}
	state := &contexts.ExecutionStateContext{Finished: true}
	httpContext := &contexts.HTTPContext{}

	err := pollTask(core.ActionHookContext{
		RunnerTasks: tasks,
		Parameters: map[string]any{
			"task_id":         "task-1",
			"organization_id": "org-1",
		},
		HTTP:           httpContext,
		ExecutionState: state,
		Usage:          recorder,
		Configuration:  hostedClaudeConfiguration(),
		Logger:         log.NewEntry(log.New()),
		Requests:       &contexts.RequestContext{},
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	require.Len(t, recorder.records, 1)
	assert.Equal(t, int64(1280), recorder.records[0].TotalTokens)
	require.Len(t, recorder.computes, 1)
	assert.Empty(t, state.Payloads)
	assert.True(t, state.Finished)
}

func TestPollTaskKeepsPollingWhenFinishedAndBrokerNotTerminal(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-1", State: "running"}}

	recorder := &recordingUsage{}
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{Finished: true}
	httpContext := &contexts.HTTPContext{}

	err := pollTask(core.ActionHookContext{
		RunnerTasks: tasks,
		Parameters: map[string]any{
			"task_id":         "task-1",
			"organization_id": "org-1",
		},
		HTTP:           httpContext,
		ExecutionState: state,
		Usage:          recorder,
		Configuration:  hostedClaudeConfiguration(),
		Logger:         log.NewEntry(log.New()),
		Requests:       requests,
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	assert.Empty(t, recorder.records)
	assert.Empty(t, recorder.computes)
	assert.Equal(t, hookActionPoll, requests.Action)
	assert.Equal(t, "task-1", requests.Params["task_id"])
	assert.Equal(t, "org-1", requests.Params["organization_id"])
	assert.Equal(t, pollInterval, requests.Duration)
}

func TestPollTaskKeepsPollingWhenFinishedAndFetchFails(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{FindErr: errors.New("task unavailable")}

	recorder := &recordingUsage{}
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{Finished: true}
	httpContext := &contexts.HTTPContext{}

	err := pollTask(core.ActionHookContext{
		RunnerTasks: tasks,
		Parameters: map[string]any{
			"task_id":         "task-1",
			"organization_id": "org-1",
		},
		HTTP:           httpContext,
		ExecutionState: state,
		Usage:          recorder,
		Configuration:  hostedClaudeConfiguration(),
		Logger:         log.NewEntry(log.New()),
		Requests:       requests,
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	assert.Empty(t, recorder.records)
	assert.Equal(t, hookActionPoll, requests.Action)
	assert.Equal(t, "task-1", requests.Params["task_id"])
}

func TestCancelTaskRecordsUsageWhenTaskAlreadyTerminal(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: canceledTaskWithUsage()}

	recorder := &recordingUsage{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{"task_id": "task-1"}}
	httpContext := &contexts.HTTPContext{}

	err := cancelTask(core.ExecutionContext{
		RunnerTasks:    tasks,
		HTTP:           httpContext,
		ExecutionState: state,
		Usage:          recorder,
		Configuration:  hostedClaudeConfiguration(),
		Logger:         log.NewEntry(log.New()),
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	require.Len(t, recorder.records, 1)
	assert.Equal(t, int64(1280), recorder.records[0].TotalTokens)
	require.Len(t, recorder.computes, 1)
	assert.Equal(t, FailedOutputChannel, state.Channel)
	assert.False(t, state.Cancelled)
}

func TestCancelTaskCancelsAnalysisWhenTaskAlreadyTerminal(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: canceledTaskWithUsage()}

	recorder := &recordingUsage{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{"task_id": "task-1"}}
	markAnalysisSession(state)
	httpContext := &contexts.HTTPContext{}

	err := cancelTask(core.ExecutionContext{
		RunnerTasks:    tasks,
		HTTP:           httpContext,
		ExecutionState: state,
		Usage:          recorder,
		Configuration:  hostedClaudeConfiguration(),
		Logger:         log.NewEntry(log.New()),
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	require.Len(t, recorder.records, 1)
	assert.Equal(t, int64(1280), recorder.records[0].TotalTokens)
	require.Len(t, recorder.computes, 1)
	assert.True(t, state.Cancelled)
	assert.Empty(t, state.Channel)
}

func TestCancelTaskSchedulesPollWhenTaskNotTerminal(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-1", State: "running"}}

	recorder := &recordingUsage{}
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{"task_id": "task-1"}}
	httpContext := &contexts.HTTPContext{}

	err := cancelTask(core.ExecutionContext{
		RunnerTasks:    tasks,
		HTTP:           httpContext,
		ExecutionState: state,
		Usage:          recorder,
		Configuration:  hostedClaudeConfiguration(),
		Logger:         log.NewEntry(log.New()),
		Requests:       requests,
		OrganizationID: "org-1",
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	assert.Empty(t, recorder.records)
	assert.Empty(t, recorder.computes)
	assert.Empty(t, state.Channel)
	assert.Equal(t, hookActionPoll, requests.Action)
	assert.Equal(t, "task-1", requests.Params["task_id"])
	assert.Equal(t, "org-1", requests.Params["organization_id"])
	assert.Equal(t, pollInterval, requests.Duration)
}

func hostedClaudeConfiguration() map[string]any {
	return map[string]any{
		"credentials": map[string]any{"source": "hosted"},
		"model":       "sonnet",
	}
}

func canceledTaskWithUsage() *core.RunnerTask {
	started := time.Now().Add(-2 * time.Second)
	finished := time.Now()
	exitCode := 130
	return &core.RunnerTask{ID: "task-1", State: "canceled", ExitCode: &exitCode, StartedAt: &started, FinishedAt: &finished,
		Result: json.RawMessage(`{"usage":{"input_tokens":1200,"output_tokens":80},"model":"claude-sonnet-4-6"}`)}
}

func TestPollTaskFinishesLostTask(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-lost", State: "lost"}}
	state := &contexts.ExecutionStateContext{}
	require.NoError(t, pollTask(core.ActionHookContext{
		RunnerTasks: tasks, ExecutionState: state, Parameters: map[string]any{"task_id": "task-lost"},
	}, "runner.finished"))
	assert.Equal(t, FailedOutputChannel, state.Channel)
	require.Len(t, state.Payloads, 1)
	payload := state.Payloads[0].(map[string]any)
	result := payload["data"].(map[string]any)
	assert.Equal(t, "failed", result["status"])
	assert.Equal(t, "runner task was lost", result["error"])
}
