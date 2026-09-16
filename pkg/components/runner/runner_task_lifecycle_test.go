package runner

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
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

func TestPollBrokerTaskRecordsUsageWhenSuperPlaneAlreadyFinished(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	recorder := &recordingUsage{}
	state := &contexts.ExecutionStateContext{Finished: true}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{canceledBrokerTaskWithUsageResponse()}}

	err := pollBrokerTask(core.ActionHookContext{
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

func TestPollBrokerTaskKeepsPollingWhenFinishedAndBrokerNotTerminal(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	recorder := &recordingUsage{}
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{Finished: true}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{runningBrokerTaskResponse()}}

	err := pollBrokerTask(core.ActionHookContext{
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

func TestPollBrokerTaskKeepsPollingWhenFinishedAndFetchFails(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	recorder := &recordingUsage{}
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{Finished: true}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`error`))},
	}}

	err := pollBrokerTask(core.ActionHookContext{
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

func TestCancelBrokerTaskRecordsUsageWhenBrokerAlreadyTerminal(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	recorder := &recordingUsage{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{"task_id": "task-1"}}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
		canceledBrokerTaskWithUsageResponse(),
	}}

	err := cancelBrokerTask(core.ExecutionContext{
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
}

func TestCancelBrokerTaskSchedulesPollWhenBrokerNotTerminal(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	recorder := &recordingUsage{}
	requests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{"task_id": "task-1"}}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
		runningBrokerTaskResponse(),
	}}

	err := cancelBrokerTask(core.ExecutionContext{
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

func runningBrokerTaskResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"task-1","status":"running"}`)),
	}
}

func canceledBrokerTaskWithUsageResponse() *http.Response {
	claimed := time.Now().Add(-2 * time.Second).UTC().Format(time.RFC3339Nano)
	finished := time.Now().UTC().Format(time.RFC3339Nano)
	body := fmt.Sprintf(
		`{"id":"task-1","status":"canceled","exit_code":130,"claimed_at":%q,"finished_at":%q,"result":{"usage":{"input_tokens":1200,"output_tokens":80},"model":"claude-sonnet-4-6"}}`,
		claimed,
		finished,
	)
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}
