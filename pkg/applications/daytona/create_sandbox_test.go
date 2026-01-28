package daytona

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateSandbox__Setup(t *testing.T) {
	component := CreateSandbox{}

	t.Run("valid setup with no configuration", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup with all fields", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"snapshot":         "default",
				"target":           "us",
				"autoStopInterval": 15,
			},
		})

		require.NoError(t, err)
	})

	t.Run("negative autoStopInterval -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"autoStopInterval": -1,
			},
		})

		require.ErrorContains(t, err, "autoStopInterval must be a positive number")
	})
}

func Test__CreateSandbox__Execute(t *testing.T) {
	component := CreateSandbox{}

	t.Run("successful sandbox creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-123","state":"started"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"target": "us",
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, SandboxPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("sandbox creation with snapshot", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"sandbox-456","state":"started"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"snapshot":         "custom-snapshot",
				"autoStopInterval": 30,
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:   map[string]any{},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create sandbox")
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
	assert.Len(t, config, 3)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "snapshot")
	assert.Contains(t, fieldNames, "target")
	assert.Contains(t, fieldNames, "autoStopInterval")

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
