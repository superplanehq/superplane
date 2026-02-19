package flyio

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const (
	machineJSON = `{
		"id": "machine-abc123",
		"name": "test-machine",
		"state": "%s",
		"region": "iad",
		"config": {"image": "registry.fly.io/my-app:latest"}
	}`
)

// buildMachineResp is a helper to build a mock HTTP machine response for a given state.
func buildMachineResp(state string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(replaceMachineState(state))),
	}
}

func replaceMachineState(state string) string {
	return `{"id":"machine-abc123","name":"test-machine","state":"` + state + `","region":"iad","config":{"image":"registry.fly.io/my-app:latest"}}`
}

// ---------- StartMachine ----------

func Test__StartMachine__Execute__SchedulesPoll(t *testing.T) {
	c := &StartMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			// StartMachine POST
			{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(""))},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       requests,
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	// Execution must NOT be finished yet — it's async
	assert.False(t, executionState.Finished)

	// A poll action must have been scheduled
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, startMachinePollInterval, requests.Duration)

	// Verify the start request was sent correctly
	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.String(), "/v1/apps/my-fly-app/machines/machine-abc123/start")
}

func Test__StartMachine__Poll__EmitsSuccessWhenStarted(t *testing.T) {
	c := &StartMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			buildMachineResp("started"),
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ActionContext{
		Name:           "poll",
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       requests,
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	require.True(t, executionState.Finished)
	require.True(t, executionState.Passed)
	assert.Equal(t, StartMachineSuccessOutputChannel, executionState.Channel)
	assert.Equal(t, StartMachinePayloadType, executionState.Type)

	require.Len(t, executionState.Payloads, 1)
	data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "machine-abc123", data["machineId"])
	assert.Equal(t, "my-fly-app", data["appName"])
	assert.Equal(t, "started", data["state"])
	assert.Equal(t, "iad", data["region"])
}

func Test__StartMachine__Poll__EmitsFailedWhenStopped(t *testing.T) {
	c := &StartMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			buildMachineResp("stopped"),
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ActionContext{
		Name:           "poll",
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       &contexts.RequestContext{},
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	require.True(t, executionState.Finished)
	assert.Equal(t, StartMachineFailedOutputChannel, executionState.Channel)
}

func Test__StartMachine__Poll__ReschedulesWhenTransitioning(t *testing.T) {
	c := &StartMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			buildMachineResp("starting"),
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ActionContext{
		Name:           "poll",
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       requests,
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	// Still not finished — waiting for stable state
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requests.Action)
}

func Test__StartMachine__Poll__ReschedulesOnAPIError(t *testing.T) {
	c := &StartMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: 500, Body: io.NopCloser(bytes.NewBufferString(`{"message":"internal error"}`))},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ActionContext{
		Name:           "poll",
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       requests,
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	// Transient errors should reschedule, not fail
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requests.Action)
}

func Test__StartMachine__Execute__MissingApp(t *testing.T) {
	c := &StartMachine{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "t"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Requests:       &contexts.RequestContext{},
		Configuration:  map[string]any{"machine": "x/y"},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is required")
}

func Test__StartMachine__Execute__MissingMachine(t *testing.T) {
	c := &StartMachine{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "t"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Requests:       &contexts.RequestContext{},
		Configuration:  map[string]any{"app": "my-fly-app"},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "machine is required")
}
