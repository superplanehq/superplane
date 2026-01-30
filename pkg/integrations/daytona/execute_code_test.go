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

	t.Run("successful code execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"exitCode":0,"result":"hello world"}`)),
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
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "print('hello world')",
				"language":  "python",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ExecuteCodePayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("code execution with non-zero exit code", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"proxyToolboxUrl":"https://app.daytona.io/api/toolbox"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"exitCode":1,"result":"error: division by zero"}`)),
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
			Configuration: map[string]any{
				"sandboxId": "sandbox-123",
				"code":      "print(1/0)",
				"language":  "python",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
	})

	t.Run("code execution failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"sandbox not found"}`)),
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
			Configuration: map[string]any{
				"sandboxId": "invalid-sandbox",
				"code":      "print('hello')",
				"language":  "python",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute code")
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
