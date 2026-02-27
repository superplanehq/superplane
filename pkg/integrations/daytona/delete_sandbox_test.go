package daytona

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteSandbox__Setup(t *testing.T) {
	component := DeleteSandbox{}

	t.Run("sandbox is required", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandbox": "",
			},
		})

		require.ErrorContains(t, err, "sandbox is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup with force flag", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"sandbox": "sandbox-123",
				"force":   true,
			},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteSandbox__Execute(t *testing.T) {
	component := DeleteSandbox{}

	t.Run("successful sandbox deletion", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
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
				"sandbox": "sandbox-123",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, DeleteSandboxPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("successful force deletion", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
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
				"sandbox": "sandbox-123",
				"force":   true,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "force=true")
	})

	t.Run("sandbox deletion failure -> error", func(t *testing.T) {
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
				"sandbox": "invalid-sandbox",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete sandbox")
	})
}

func Test__DeleteSandbox__Configuration(t *testing.T) {
	component := DeleteSandbox{}

	config := component.Configuration()
	require.Len(t, config, 2)

	var sandboxField *configuration.Field
	for i := range config {
		if config[i].Name == "sandbox" {
			sandboxField = &config[i]
			break
		}
	}

	require.NotNil(t, sandboxField)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, sandboxField.Type)
	require.NotNil(t, sandboxField.TypeOptions)
	require.NotNil(t, sandboxField.TypeOptions.Resource)
	assert.Equal(t, "sandbox", sandboxField.TypeOptions.Resource.Type)
}
