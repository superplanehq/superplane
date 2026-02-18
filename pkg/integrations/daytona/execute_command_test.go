package daytona

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

	t.Run("successful execution -> creates session and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"sp-test"}`))},
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"cmdId":"cmd-abc"}`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}
		execCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"command":   "echo hello world",
				"timeout":   60,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
			Metadata:       metadataCtx,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.False(t, execCtx.Finished)
		assert.Equal(t, "poll", requestsCtx.Action)
		assert.Equal(t, ExecuteCommandPollInterval, requestsCtx.Duration)

		metadata, ok := metadataCtx.Metadata.(ExecuteCommandMetadata)
		require.True(t, ok)
		assert.Equal(t, "sandbox-123", metadata.SandboxID)
		assert.Equal(t, "cmd-abc", metadata.CmdID)
		assert.NotEmpty(t, metadata.SessionID)
		assert.NotZero(t, metadata.StartedAt)
	})

	t.Run("session creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"internal error"}`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"command":   "echo hello",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create session")
	})
}

func Test__ExecuteCommand__HandleAction(t *testing.T) {
	component := ExecuteCommand{}

	t.Run("unknown action -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{Name: "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})

	t.Run("already finished -> no-op", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			ExecutionState: &contexts.ExecutionStateContext{Finished: true, KVs: map[string]string{}},
		})

		require.NoError(t, err)
	})

	t.Run("command still running -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"s-1","commands":[{"cmdId":"cmd-abc","command":"echo hi","exitCode":null}]}`))},
			},
		}

		requestsCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: ExecuteCommandMetadata{
					SandboxID: "sandbox-123",
					SessionID: "s-1",
					CmdID:     "cmd-abc",
					StartedAt: time.Now().Unix(),
					Timeout:   600,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestsCtx.Action)
		assert.Equal(t, ExecuteCommandPollInterval, requestsCtx.Duration)
	})

	t.Run("command completed -> emits result", func(t *testing.T) {
		exitCode := 0
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"s-1","commands":[{"cmdId":"cmd-abc","command":"echo hi","exitCode":0}]}`))},
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`"hello world"`))},
			},
		}

		_ = exitCode
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: ExecuteCommandMetadata{
					SandboxID: "sandbox-123",
					SessionID: "s-1",
					CmdID:     "cmd-abc",
					StartedAt: time.Now().Unix(),
					Timeout:   600,
				},
			},
			ExecutionState: execState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.True(t, execState.Passed)
		assert.Equal(t, ExecuteCommandPayloadType, execState.Type)
	})

	t.Run("timeout exceeded -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: ExecuteCommandMetadata{
					SandboxID: "sandbox-123",
					SessionID: "s-1",
					CmdID:     "cmd-abc",
					StartedAt: time.Now().Add(-10 * time.Minute).Unix(),
					Timeout:   60,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})

	t.Run("get session error -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"error"}`))},
			},
		}

		requestsCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: ExecuteCommandMetadata{
					SandboxID: "sandbox-123",
					SessionID: "s-1",
					CmdID:     "cmd-abc",
					StartedAt: time.Now().Unix(),
					Timeout:   600,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestsCtx.Action)
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

func Test__ExecuteCommand__Actions(t *testing.T) {
	component := ExecuteCommand{}

	actions := component.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "poll", actions[0].Name)
	assert.False(t, actions[0].UserAccessible)
}
