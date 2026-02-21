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

func Test__ExecuteCommand__Setup(t *testing.T) {
	component := ExecuteCommand{}

	t.Run("sandboxId is required", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "",
				"command":   "echo hello",
			},
		})

		require.ErrorContains(t, err, "sandboxId is required")
	})

	t.Run("command is required", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"command":   "",
			},
		})

		require.ErrorContains(t, err, "command is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"command":   "echo hello",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup with optional fields", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"command":   "pip install requests",
				"cwd":       "/home/daytona",
				"timeout":   60,
			},
		})

		require.NoError(t, err)
	})
}

func Test__ExecuteCommand__Execute(t *testing.T) {
	component := ExecuteCommand{}

	t.Run("schedules poll after async kickoff", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// FetchConfig for CreateSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// CreateSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				// FetchConfig for ExecuteSessionCommand
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// ExecuteSessionCommand
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-001"}`))},
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
				"sandboxId": "sandbox-123",
				"command":   "echo hello world",
				"timeout":   120,
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
		assert.Equal(t, ExecuteCommandPollInterval, requestCtx.Duration)

		metadata, ok := metadataCtx.Metadata.(ExecuteCommandMetadata)
		require.True(t, ok)
		assert.Equal(t, "sandbox-123", metadata.SandboxID)
		assert.Equal(t, "cmd-001", metadata.CmdID)
		assert.Equal(t, 120, metadata.Timeout)
		assert.NotEmpty(t, metadata.SessionID)
	})

	t.Run("prepends cd when cwd is set", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-001"}`))},
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
				"sandboxId": "sandbox-123",
				"command":   "pwd",
				"cwd":       "/home/daytona",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// The 4th request (index 3) is the ExecuteSessionCommand call
		require.Len(t, httpContext.Requests, 4)
		body, _ := io.ReadAll(httpContext.Requests[3].Body)
		// Go's json.Marshal escapes & as \u0026, so check for both forms
		bodyStr := string(body)
		assert.True(t,
			strings.Contains(bodyStr, "cd /home/daytona && pwd") ||
				strings.Contains(bodyStr, `cd /home/daytona \u0026\u0026 pwd`),
			"expected command with cd prefix, got: %s", bodyStr,
		)
	})

	t.Run("create session failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"message":"sandbox not found"}`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sandboxId": "invalid-sandbox",
				"command":   "echo hello",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create session")
	})
}

func Test__ExecuteCommand__HandleAction(t *testing.T) {
	component := ExecuteCommand{}

	t.Run("poll reschedules when command is still running", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-abc","commands":[{"id":"cmd-001"}]}`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"sessionId": "session-abc",
					"cmdId":     "cmd-001",
					"startedAt": time.Now().Unix(),
					"timeout":   300,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("poll emits result when command completes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// FetchConfig for GetSession
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// GetSession - command completed with exit code 0
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-abc","commands":[{"id":"cmd-001","exitCode":0}]}`))},
				// FetchConfig for GetSessionCommandLogs
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				// GetSessionCommandLogs
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`hello world`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"sessionId": "session-abc",
					"cmdId":     "cmd-001",
					"startedAt": time.Now().Unix(),
					"timeout":   300,
				},
			},
			ExecutionState: execCtx,
			Requests:       &contexts.RequestContext{},
			Integration:    appCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ExecuteCommandPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("poll times out", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"sessionId": "session-abc",
					"cmdId":     "cmd-001",
					"startedAt": time.Now().Add(-10 * time.Minute).Unix(),
					"timeout":   60,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Integration:    appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})

	t.Run("poll reschedules on GetSession API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"server error"}`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"sandboxId": "sandbox-123",
					"sessionId": "session-abc",
					"cmdId":     "cmd-001",
					"startedAt": time.Now().Unix(),
					"timeout":   300,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       requestCtx,
			Integration:    appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
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

func Test__ExecuteCommand__ComponentInfo(t *testing.T) {
	component := ExecuteCommand{}

	assert.Equal(t, "daytona.executeCommand", component.Name())
	assert.Equal(t, "Execute Command", component.Label())
	assert.Equal(t, "Run a shell command in a sandbox environment", component.Description())
	assert.Equal(t, "daytona", component.Icon())
	assert.Equal(t, "orange", component.Color())
	assert.NotEmpty(t, component.Documentation())
}

func Test__ExecuteCommand__Configuration(t *testing.T) {
	component := ExecuteCommand{}

	config := component.Configuration()
	assert.Len(t, config, 4)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "sandboxId")
	assert.Contains(t, fieldNames, "command")
	assert.Contains(t, fieldNames, "cwd")
	assert.Contains(t, fieldNames, "timeout")

	for _, f := range config {
		if f.Name == "sandboxId" || f.Name == "command" {
			assert.True(t, f.Required, "%s should be required", f.Name)
		}
		if f.Name == "cwd" || f.Name == "timeout" {
			assert.False(t, f.Required, "%s should be optional", f.Name)
		}
	}
}

func Test__ExecuteCommand__OutputChannels(t *testing.T) {
	component := ExecuteCommand{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}
