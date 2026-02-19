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

const createMachineResponseJSON = `{
	"id": "new-machine-xyz",
	"name": "generated-machine",
	"state": "created",
	"region": "iad",
	"config": {"image": "registry.fly.io/my-app:v2"}
}`

func Test__CreateMachine__Execute__SchedulesPoll(t *testing.T) {
	c := &CreateMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			// CreateMachine POST response
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(createMachineResponseJSON)),
			},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}
	metadata := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       requests,
		Metadata:       metadata,
		Configuration: map[string]any{
			"app":   "my-fly-app",
			"image": "registry.fly.io/my-app:v2",
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	// Execution must NOT be finished yet — it's async
	assert.False(t, executionState.Finished)

	// A poll action must have been scheduled
	assert.Equal(t, "poll", requests.Action)
	assert.Equal(t, createMachinePollInterval, requests.Duration)

	// Machine ID must have been stored in metadata
	storedMeta, ok := metadata.Metadata.(CreateMachineExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, "new-machine-xyz", storedMeta.MachineID)

	// Verify the create request was sent correctly
	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.String(), "/v1/apps/my-fly-app/machines")
}

func Test__CreateMachine__Execute__WithAllOptions(t *testing.T) {
	c := &CreateMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(createMachineResponseJSON)),
			},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	requests := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}
	metadata := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       requests,
		Metadata:       metadata,
		Configuration: map[string]any{
			"app":      "my-fly-app",
			"image":    "registry.fly.io/my-app:v2",
			"region":   "lhr",
			"cpuKind":  "performance",
			"cpus":     2,
			"memoryMB": 512,
			"name":     "my-worker",
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requests.Action)
}

func Test__CreateMachine__Poll__EmitsSuccessWhenStarted(t *testing.T) {
	c := &CreateMachine{}

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

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ActionContext{
		Name:           "poll",
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Requests:       &contexts.RequestContext{},
		Metadata:       &contexts.MetadataContext{Metadata: CreateMachineExecutionMetadata{MachineID: "machine-abc123"}},
		Configuration: map[string]any{
			"app":   "my-fly-app",
			"image": "registry.fly.io/my-app:v2",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	require.True(t, executionState.Finished)
	require.True(t, executionState.Passed)
	assert.Equal(t, CreateMachineSuccessOutputChannel, executionState.Channel)
	assert.Equal(t, CreateMachinePayloadType, executionState.Type)

	require.Len(t, executionState.Payloads, 1)
	data := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "machine-abc123", data["machineId"])
	assert.Equal(t, "my-fly-app", data["appName"])
	assert.Equal(t, "started", data["state"])
}

func Test__CreateMachine__Poll__EmitsFailedWhenStopped(t *testing.T) {
	c := &CreateMachine{}

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
		Metadata:       &contexts.MetadataContext{Metadata: CreateMachineExecutionMetadata{MachineID: "machine-abc123"}},
		Configuration: map[string]any{
			"app":   "my-fly-app",
			"image": "registry.fly.io/my-app:v2",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	require.True(t, executionState.Finished)
	assert.Equal(t, CreateMachineFailedOutputChannel, executionState.Channel)
}

func Test__CreateMachine__Poll__ReschedulesWhenCreated(t *testing.T) {
	c := &CreateMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			buildMachineResp("created"),
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
		Metadata:       &contexts.MetadataContext{Metadata: CreateMachineExecutionMetadata{MachineID: "machine-abc123"}},
		Configuration: map[string]any{
			"app":   "my-fly-app",
			"image": "registry.fly.io/my-app:v2",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requests.Action)
}

func Test__CreateMachine__Poll__ReschedulesOnAPIError(t *testing.T) {
	c := &CreateMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: 503, Body: io.NopCloser(bytes.NewBufferString(`{"message":"service unavailable"}`))},
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
		Metadata:       &contexts.MetadataContext{Metadata: CreateMachineExecutionMetadata{MachineID: "machine-abc123"}},
		Configuration: map[string]any{
			"app":   "my-fly-app",
			"image": "registry.fly.io/my-app:v2",
		},
	}

	err := c.HandleAction(ctx)
	require.NoError(t, err)

	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requests.Action)
}

func Test__CreateMachine__Execute__MissingApp(t *testing.T) {
	c := &CreateMachine{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "t"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Requests:       &contexts.RequestContext{},
		Metadata:       &contexts.MetadataContext{},
		Configuration:  map[string]any{"image": "some-image"},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app is required")
}

func Test__CreateMachine__Execute__MissingImage(t *testing.T) {
	c := &CreateMachine{}

	ctx := core.ExecutionContext{
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "t"}},
		ExecutionState: &contexts.ExecutionStateContext{},
		Requests:       &contexts.RequestContext{},
		Metadata:       &contexts.MetadataContext{},
		Configuration:  map[string]any{"app": "my-fly-app"},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image is required")
}

func Test__CreateMachine__Execute__APIError(t *testing.T) {
	c := &CreateMachine{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: 422, Body: io.NopCloser(bytes.NewBufferString(`{"message":"invalid image"}`))},
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
		Requests:       &contexts.RequestContext{},
		Metadata:       &contexts.MetadataContext{},
		Configuration: map[string]any{
			"app":   "my-fly-app",
			"image": "bad-image",
		},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create machine")
}
