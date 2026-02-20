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

func Test__DeleteMachine__Execute__Success(t *testing.T) {
	c := &DeleteMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			// DELETE returns 200 with empty body
			{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(""))},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Logger:         testLogger(),
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	// Verify HTTP request
	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Equal(t, http.MethodDelete, req.Method)
	assert.Contains(t, req.URL.String(), "/v1/apps/my-fly-app/machines/machine-abc123")
	assert.NotContains(t, req.URL.String(), "force=true")
	assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

	// Verify output
	require.True(t, executionState.Finished)
	require.True(t, executionState.Passed)
	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, DeleteMachinePayloadType, executionState.Type)

	require.Len(t, executionState.Payloads, 1)
	data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "machine-abc123", data["machineId"])
	assert.Equal(t, "my-fly-app", data["appName"])
	assert.Equal(t, true, data["deleted"])
}

func Test__DeleteMachine__Execute__ForceFlag(t *testing.T) {
	c := &DeleteMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(""))},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Logger:         testLogger(),
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
			"force":   true,
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Equal(t, http.MethodDelete, req.Method)
	assert.Contains(t, req.URL.String(), "force=true")

	require.True(t, executionState.Finished)
	require.True(t, executionState.Passed)
}

func Test__DeleteMachine__Execute__APIError(t *testing.T) {
	c := &DeleteMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: 404, Body: io.NopCloser(bytes.NewBufferString(`{"message":"machine not found"}`))},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: &contexts.ExecutionStateContext{},
		Logger:         testLogger(),
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/nonexistent",
		},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete machine")
}

func Test__DeleteMachine__Execute__MissingApp(t *testing.T) {
	c := &DeleteMachine{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "t"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Configuration:  map[string]any{"machine": "x/y"},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is required")
}

func Test__DeleteMachine__Execute__MissingMachine(t *testing.T) {
	c := &DeleteMachine{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "t"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Configuration:  map[string]any{"app": "my-fly-app"},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "machine is required")
}

func Test__DeleteMachine__Setup__Valid(t *testing.T) {
	c := &DeleteMachine{}
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{
			"app":     "my-fly-app",
			"machine": "my-fly-app/machine-abc123",
		},
	})
	assert.NoError(t, err)
}
