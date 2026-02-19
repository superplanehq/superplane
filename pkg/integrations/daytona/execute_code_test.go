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

	t.Run("schedules poll after async kickoff", func(t *testing.T) {
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
				"code":      "print('hello world')",
				"language":  "python",
				"timeout":   60,
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
		assert.Equal(t, ExecuteCodePollInterval, requestCtx.Duration)

		metadata, ok := metadataCtx.Metadata.(ExecuteCodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "sandbox-123", metadata.SandboxID)
		assert.Equal(t, "cmd-001", metadata.CmdID)
		assert.Equal(t, 60, metadata.Timeout)
	})

	t.Run("constructs python command correctly", func(t *testing.T) {
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

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "print(42)",
				"language":  "python",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 4)
		body, _ := io.ReadAll(httpContext.Requests[3].Body)
		assert.Contains(t, string(body), "python3 -c")
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
				"code":      "print('hello')",
				"language":  "python",
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

func Test__ExecuteCode__HandleAction(t *testing.T) {
	component := ExecuteCode{}

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

	t.Run("poll emits result when command completes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"sessionId":"session-abc","commands":[{"id":"cmd-001","exitCode":0}]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`42`))},
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
		assert.Equal(t, ExecuteCodePayloadType, execCtx.Type)
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
					"timeout":   30,
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Integration:    appCtx,
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
