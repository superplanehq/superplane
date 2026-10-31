package runner

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
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

func TestPollTaskExposesCodexFailureCause(t *testing.T) {
	const providerError = "Your ChatGPT subscription has reached its usage limit."
	for _, tc := range []struct {
		name              string
		finishedEventType string
		runnerError       string
		result            json.RawMessage
		expectedError     string
	}{
		{
			name:          "provider error without a runner error",
			result:        json.RawMessage(`{"type":"result","is_error":true,"error":"` + providerError + `","result":"private task output"}`),
			expectedError: providerError,
		},
		{
			name:          "provider error replaces shell exit code",
			runnerError:   "exit code 1",
			result:        json.RawMessage(`{"type":"result","is_error":true,"error":"` + providerError + `"}`),
			expectedError: providerError,
		},
		{
			name:          "provider error replaces process exit status",
			runnerError:   "exit status 1",
			result:        json.RawMessage(`{"type":"result","is_error":true,"error":"` + providerError + `"}`),
			expectedError: providerError,
		},
		{
			name:          "runner timeout takes precedence",
			runnerError:   "execution timed out",
			result:        json.RawMessage(`{"type":"result","is_error":true,"error":"` + providerError + `"}`),
			expectedError: "execution timed out",
		},
		{
			name:          "a previous successful turn does not supply an error",
			runnerError:   "exit code 1",
			result:        json.RawMessage(`{"type":"result","is_error":false,"error":"stale error"}`),
			expectedError: "exit code 1",
		},
		{
			name:              "other components keep their result as ordinary output",
			finishedEventType: RunBashFinishedEventType,
			runnerError:       "exit code 1",
			result:            json.RawMessage(`{"type":"result","is_error":true,"error":"ordinary output field"}`),
			expectedError:     "exit code 1",
		},
		{
			name: "failure without a retained cause still produces a warning",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			finishedEventType := tc.finishedEventType
			if finishedEventType == "" {
				finishedEventType = "runnerCodex.finished"
			}
			logger, hook := logtest.NewNullLogger()
			state := &contexts.ExecutionStateContext{}
			ctx := core.ActionHookContext{
				RunnerTasks: &contexts.RunnerTaskContext{Task: &core.RunnerTask{
					ID: "task-1", State: "failed", ExitCode: intPtr(1),
					ErrorMessage: tc.runnerError, Result: tc.result,
				}},
				ExecutionState: state,
				Parameters:     map[string]any{"task_id": "task-1", "organization_id": "org-1"},
				Logger: log.NewEntry(logger).WithFields(log.Fields{
					"execution": "execution-1", "root_event": "root-1",
				}),
			}

			require.NoError(t, pollTask(ctx, finishedEventType))
			assert.Equal(t, FailedOutputChannel, state.Channel)
			require.Len(t, state.Payloads, 1)
			payload := state.Payloads[0].(map[string]any)["data"].(map[string]any)
			if tc.expectedError == "" {
				assert.NotContains(t, payload, "error")
			} else {
				assert.Equal(t, tc.expectedError, payload["error"])
			}

			require.Len(t, hook.AllEntries(), 1)
			entry := hook.LastEntry()
			assert.Equal(t, log.WarnLevel, entry.Level)
			assert.Equal(t, "runner.task", entry.Data["component"])
			assert.Equal(t, "org-1", entry.Data["organization_id"])
			assert.Equal(t, "task-1", entry.Data["task_id"])
			assert.Equal(t, "execution-1", entry.Data["execution"])
			assert.Equal(t, "root-1", entry.Data["root_event"])
			assert.Equal(t, "failed", entry.Data["status"])
			assert.Equal(t, 1, entry.Data["exit_code"])
			if tc.expectedError == "" {
				assert.NotContains(t, entry.Data, "error")
			} else {
				assert.Equal(t, tc.expectedError, entry.Data["error"])
			}
			assert.NotContains(t, entry.Data, "result")
			logged, err := json.Marshal(entry.Data)
			require.NoError(t, err)
			assert.NotContains(t, string(logged), "private task output")

			require.NoError(t, pollTask(ctx, finishedEventType))
			assert.Len(t, hook.AllEntries(), 1, "completed execution must not log the failure again")
		})
	}
}

func TestFailedCodexOutputPreservesQuotaAndMeasuredUsage(t *testing.T) {
	state := &contexts.ExecutionStateContext{}
	result := json.RawMessage(`{"type":"result","is_error":true,"failure":{"reason":"chatgpt_quota_exhausted","limit_id":"codex","subscription_owner_id":"a8d2fa49-cf1f-4a3d-a65b-3205af2849df","untrusted":"discard"},"usage":{"input_tokens":8,"cache_read_input_tokens":2,"total_tokens":10},"usage_state":"partial","telemetry":{"usage_state":"partial"}}`)
	require.NoError(t, pollTask(core.ActionHookContext{
		RunnerTasks:    &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-quota", State: "failed", ExitCode: intPtr(1), Result: result}},
		ExecutionState: state, Parameters: map[string]any{"task_id": "task-quota"},
	}, "runnerCodex.finished"))
	require.Len(t, state.Payloads, 1)
	output := state.Payloads[0].(map[string]any)["data"].(map[string]any)
	failure, ok := output["failure"].(*models.SubscriptionFailure)
	require.True(t, ok)
	assert.Equal(t, models.SubscriptionFailureChatGPTQuotaExhausted, failure.Reason)
	assert.Equal(t, "a8d2fa49-cf1f-4a3d-a65b-3205af2849df", failure.SubscriptionOwnerID)
	persisted := output["result"].(map[string]any)
	assert.Equal(t, "partial", persisted["usage_state"])
	assert.Equal(t, float64(10), persisted["usage"].(map[string]any)["total_tokens"])
}
