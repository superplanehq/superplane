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

func Test__ExecuteCode__Setup(t *testing.T) {
	component := ExecuteCode{}

	t.Run("sandboxId is required", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "",
				"code":      "print('hello')",
				"language":  "python",
			},
		})

		require.ErrorContains(t, err, "sandboxId is required")
	})

	t.Run("code is required", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "",
				"language":  "python",
			},
		})

		require.ErrorContains(t, err, "code is required")
	})

	t.Run("language is required", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "print('hello')",
				"language":  "",
			},
		})

		require.ErrorContains(t, err, "language is required")
	})

	t.Run("invalid language -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "print('hello')",
				"language":  "ruby",
			},
		})

		require.ErrorContains(t, err, "invalid language")
		require.ErrorContains(t, err, "ruby")
	})

	t.Run("valid python setup", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "print('hello')",
				"language":  "python",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid typescript setup", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "console.log('hello')",
				"language":  "typescript",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid javascript setup", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "console.log('hello')",
				"language":  "javascript",
			},
		})

		require.NoError(t, err)
	})
}

func Test__ExecuteCode__Execute(t *testing.T) {
	component := ExecuteCode{}

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
				"code":      "print('hello world')",
				"language":  "python",
				"timeout":   30,
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
		assert.Equal(t, ExecuteCodePollInterval, requestsCtx.Duration)

		metadata, ok := metadataCtx.Metadata.(ExecuteCodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "sandbox-123", metadata.SandboxID)
		assert.Equal(t, "cmd-abc", metadata.CmdID)
		assert.NotEmpty(t, metadata.SessionID)
	})

	t.Run("session creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"message":"sandbox not found"}`))},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"sandboxId": "invalid-sandbox",
				"code":      "print('hello')",
				"language":  "python",
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

func Test__ExecuteCode__HandleAction(t *testing.T) {
	component := ExecuteCode{}

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
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"s-1","commands":[{"cmdId":"cmd-abc","command":"python3 -c ...","exitCode":null}]}`))},
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
				Metadata: ExecuteCodeMetadata{
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
		assert.Equal(t, ExecuteCodePollInterval, requestsCtx.Duration)
	})

	t.Run("execution completed -> emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"s-1","commands":[{"cmdId":"cmd-abc","command":"python3 -c ...","exitCode":0}]}`))},
				configResponse(),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`"hello world"`))},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-api-key"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: ExecuteCodeMetadata{
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
		assert.Equal(t, ExecuteCodePayloadType, execState.Type)
	})

	t.Run("timeout exceeded -> error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: ExecuteCodeMetadata{
					SandboxID: "sandbox-123",
					SessionID: "s-1",
					CmdID:     "cmd-abc",
					StartedAt: time.Now().Add(-10 * time.Minute).Unix(),
					Timeout:   30,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}

func Test__ExecuteCode__ComponentInfo(t *testing.T) {
	component := ExecuteCode{}

	assert.Equal(t, "daytona.executeCode", component.Name())
	assert.Equal(t, "Execute Code", component.Label())
	assert.Equal(t, "Execute code in a sandbox environment", component.Description())
	assert.Equal(t, "daytona", component.Icon())
	assert.Equal(t, "orange", component.Color())
	assert.NotEmpty(t, component.Documentation())
}

func Test__ExecuteCode__Configuration(t *testing.T) {
	component := ExecuteCode{}

	config := component.Configuration()
	assert.Len(t, config, 4)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "sandboxId")
	assert.Contains(t, fieldNames, "code")
	assert.Contains(t, fieldNames, "language")
	assert.Contains(t, fieldNames, "timeout")

	for _, f := range config {
		if f.Name == "sandboxId" || f.Name == "code" || f.Name == "language" {
			assert.True(t, f.Required, "%s should be required", f.Name)
		}
		if f.Name == "timeout" {
			assert.False(t, f.Required, "timeout should be optional")
		}
	}
}

func Test__ExecuteCode__OutputChannels(t *testing.T) {
	component := ExecuteCode{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__ExecuteCode__Actions(t *testing.T) {
	component := ExecuteCode{}

	actions := component.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "poll", actions[0].Name)
	assert.False(t, actions[0].UserAccessible)
}
