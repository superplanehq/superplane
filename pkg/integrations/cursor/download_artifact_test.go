package cursor

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DownloadArtifact__Execute(t *testing.T) {
	c := &DownloadArtifact{}

	t.Run("success -> emits download URL", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"url": "https://cloud-agent-artifacts.s3.us-east-1.amazonaws.com/artifacts/screenshot.png?X-Amz-Expires=900",
						"expiresAt": "2026-04-13T19:00:00.000Z"
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-agent-key",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"agent": "bc-00000000-0000-0000-0000-000000000001",
				"path":  "artifacts/screenshot.png",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t,
			"https://api.cursor.com/v1/agents/bc-00000000-0000-0000-0000-000000000001/artifacts/download?path=artifacts%2Fscreenshot.png",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t, "Bearer test-agent-key", httpContext.Requests[0].Header.Get("Authorization"))

		assert.Equal(t, core.DefaultOutputChannel.Name, executionStateCtx.Channel)
		assert.Equal(t, DownloadArtifactPayloadType, executionStateCtx.Type)

		require.Len(t, executionStateCtx.Payloads, 1)
		payload, ok := executionStateCtx.Payloads[0].(map[string]any)
		require.True(t, ok)

		output, ok := payload["data"].(DownloadArtifactOutput)
		require.True(t, ok)
		assert.Equal(t, "bc-00000000-0000-0000-0000-000000000001", output.AgentID)
		assert.Equal(t, "artifacts/screenshot.png", output.Path)
		assert.Equal(t, "https://cloud-agent-artifacts.s3.us-east-1.amazonaws.com/artifacts/screenshot.png?X-Amz-Expires=900", output.URL)
		assert.Equal(t, "2026-04-13T19:00:00.000Z", output.ExpiresAt)
	})

	t.Run("missing agent -> error", func(t *testing.T) {
		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"path": "artifacts/screenshot.png",
			},
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"launchAgentKey": "test-agent-key"}},
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("missing path -> error", func(t *testing.T) {
		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"agent": "bc-00000000-0000-0000-0000-000000000001",
			},
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"launchAgentKey": "test-agent-key"}},
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "artifact path is required")
	})

	t.Run("missing cloud agent key -> error", func(t *testing.T) {
		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"agent": "bc-00000000-0000-0000-0000-000000000001",
				"path":  "artifacts/screenshot.png",
			},
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{}},
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud agent API key is not configured")
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"artifact not found"}`)),
				},
			},
		}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"agent": "bc-00000000-0000-0000-0000-000000000001",
				"path":  "artifacts/missing.png",
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"launchAgentKey": "test-agent-key"}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download artifact")
	})
}

func Test__DownloadArtifact__Setup(t *testing.T) {
	c := &DownloadArtifact{}

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agent": "bc-00000000-0000-0000-0000-000000000001",
				"path":  "artifacts/screenshot.png",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing agent -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"path": "artifacts/screenshot.png",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("missing path -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agent": "bc-00000000-0000-0000-0000-000000000001",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "artifact path is required")
	})
}

func Test__DownloadArtifact__OutputChannels(t *testing.T) {
	c := &DownloadArtifact{}
	channels := c.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__DownloadArtifact__Configuration(t *testing.T) {
	c := &DownloadArtifact{}
	fields := c.Configuration()

	require.Len(t, fields, 2)

	agent := fields[0]
	assert.Equal(t, "agent", agent.Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, agent.Type)
	assert.True(t, agent.Required)
	require.NotNil(t, agent.TypeOptions)
	require.NotNil(t, agent.TypeOptions.Resource)
	assert.Equal(t, "agent", agent.TypeOptions.Resource.Type)

	path := fields[1]
	assert.Equal(t, "path", path.Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, path.Type)
	assert.True(t, path.Required)
	require.NotNil(t, path.TypeOptions)
	require.NotNil(t, path.TypeOptions.Resource)
	assert.Equal(t, "artifact", path.TypeOptions.Resource.Type)
	require.Len(t, path.TypeOptions.Resource.Parameters, 1)
	assert.Equal(t, "agent", path.TypeOptions.Resource.Parameters[0].Name)
	require.NotNil(t, path.TypeOptions.Resource.Parameters[0].ValueFrom)
	assert.Equal(t, "agent", path.TypeOptions.Resource.Parameters[0].ValueFrom.Field)
}

func Test__DownloadArtifact__ExampleOutput(t *testing.T) {
	c := &DownloadArtifact{}
	example := c.ExampleOutput()

	require.Contains(t, example, "data")
	require.Contains(t, example, "timestamp")
	require.Contains(t, example, "type")
	assert.Equal(t, DownloadArtifactPayloadType, example["type"])

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, data, "agentId")
	assert.Contains(t, data, "path")
	assert.Contains(t, data, "url")
	assert.Contains(t, data, "expiresAt")
}
