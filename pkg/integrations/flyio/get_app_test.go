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

func Test__GetApp__Execute__Success(t *testing.T) {
	c := &GetApp{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"id": "app-abc123",
					"name": "my-fly-app",
					"status": "deployed",
					"machine_count": 3,
					"volume_count": 1,
					"network": "default"
				}`)),
			},
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
		Configuration: map[string]any{
			"app": "my-fly-app",
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	// Verify HTTP request
	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Equal(t, http.MethodGet, req.Method)
	assert.Contains(t, req.URL.String(), "/v1/apps/my-fly-app")
	assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

	// Verify output
	require.True(t, executionState.Finished)
	require.True(t, executionState.Passed)
	assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	assert.Equal(t, GetAppPayloadType, executionState.Type)

	require.Len(t, executionState.Payloads, 1)
	data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "my-fly-app", data["name"])
	assert.Equal(t, "app-abc123", data["id"])
	assert.Equal(t, "deployed", data["status"])
	assert.Equal(t, 3, data["machineCount"])
	assert.Equal(t, 1, data["volumeCount"])
	assert.Equal(t, "default", data["network"])
}

func Test__GetApp__Execute__APIError(t *testing.T) {
	c := &GetApp{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 404,
				Body:       io.NopCloser(bytes.NewBufferString(`{"message": "app not found"}`)),
			},
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
		Configuration: map[string]any{
			"app": "nonexistent-app",
		},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get app")
}

func Test__GetApp__Execute__MissingApp(t *testing.T) {
	c := &GetApp{}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Configuration:  map[string]any{},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is required")
}

func Test__GetApp__Setup__MissingApp(t *testing.T) {
	c := &GetApp{}
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is required")
}

func Test__GetApp__Setup__Valid(t *testing.T) {
	c := &GetApp{}
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{
			"app": "my-fly-app",
		},
	})
	assert.NoError(t, err)
}
