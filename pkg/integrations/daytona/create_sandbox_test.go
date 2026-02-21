package daytona

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateSandbox__Setup(t *testing.T) {
	component := CreateSandbox{}

	t.Run("valid setup with no configuration", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   appCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup with all fields", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"snapshot":         "default",
				"target":           "us",
				"autoStopInterval": 15,
			},
		})

		require.NoError(t, err)
	})

	t.Run("negative autoStopInterval -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"autoStopInterval": -1,
			},
		})

		require.ErrorContains(t, err, "autoStopInterval cannot be negative")
	})
}

func Test__CreateSandbox__Execute(t *testing.T) {
	component := CreateSandbox{}

	t.Run("schedules poll after sandbox creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"creating"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"target": "us",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.False(t, execCtx.Finished, "execution should not be finished yet")
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, CreateSandboxPollInterval, requestCtx.Duration)

		metadata, ok := metadataCtx.Metadata.(CreateSandboxMetadata)
		require.True(t, ok)
		assert.Equal(t, "sandbox-123", metadata.SandboxID)
		assert.NotZero(t, metadata.StartedAt)
	})

	t.Run("sandbox creation with env variables schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-789","state":"creating"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		requestCtx := &contexts.RequestContext{}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"env": []map[string]any{
					{"name": "API_KEY", "value": "secret123"},
					{"name": "DEBUG", "value": "true"},
				},
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
			Metadata:       &contexts.MetadataContext{},
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.False(t, execCtx.Finished)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("sandbox creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid request"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create sandbox")
	})
}

func Test__CreateSandbox__HandleAction(t *testing.T) {
	component := CreateSandbox{}

	t.Run("poll reschedules when sandbox is still creating", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"creating"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:        "poll",
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"startedAt": time.Now().UnixNano(),
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, CreateSandboxPollInterval, requestCtx.Duration)
	})

	t.Run("poll emits result when sandbox is started", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"started"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleAction(core.ActionContext{
			Name:        "poll",
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"startedAt": time.Now().UnixNano(),
				},
			},
			ExecutionState: execCtx,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, SandboxPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("poll errors when sandbox state is error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"error"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:        "poll",
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"startedAt": time.Now().UnixNano(),
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start")
	})

	t.Run("poll reschedules on GetSandbox API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message":"server error"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name:        "poll",
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"startedAt": time.Now().UnixNano(),
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("poll times out", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name:        "poll",
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"startedAt": time.Now().Add(-10 * time.Minute).UnixNano(),
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})

	t.Run("skips when execution already finished", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		})

		require.NoError(t, err)
	})

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}

func Test__CreateSandbox__ComponentInfo(t *testing.T) {
	component := CreateSandbox{}

	assert.Equal(t, "daytona.createSandbox", component.Name())
	assert.Equal(t, "Create Sandbox", component.Label())
	assert.Equal(t, "Create an isolated sandbox environment for code execution", component.Description())
	assert.Equal(t, "daytona", component.Icon())
	assert.Equal(t, "orange", component.Color())
	assert.NotEmpty(t, component.Documentation())
}

func Test__CreateSandbox__Configuration(t *testing.T) {
	component := CreateSandbox{}

	config := component.Configuration()
	assert.Len(t, config, 4)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "snapshot")
	assert.Contains(t, fieldNames, "target")
	assert.Contains(t, fieldNames, "autoStopInterval")
	assert.Contains(t, fieldNames, "env")

	for _, f := range config {
		assert.False(t, f.Required, "all fields should be optional")
	}
}

func Test__CreateSandbox__OutputChannels(t *testing.T) {
	component := CreateSandbox{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}
