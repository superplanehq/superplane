package prometheus

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

func Test__ExpireSilence__Setup(t *testing.T) {
	component := &ExpireSilence{}

	t.Run("configuration uses silence resource field", func(t *testing.T) {
		fields := component.Configuration()
		require.Len(t, fields, 1)
		assert.Equal(t, "silence", fields[0].Name)
		assert.Equal(t, "Silence", fields[0].Label)
		assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
		require.NotNil(t, fields[0].TypeOptions)
		require.NotNil(t, fields[0].TypeOptions.Resource)
		assert.Equal(t, ResourceTypeSilence, fields[0].TypeOptions.Resource.Type)
	})

	t.Run("silence is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"silence": ""},
		})
		require.ErrorContains(t, err, "silence is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"silence": "abc123"},
		})
		require.NoError(t, err)
	})
}

func Test__ExpireSilence__Execute(t *testing.T) {
	component := &ExpireSilence{}

	t.Run("silence is expired and emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"silence": "abc123"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       metadataCtx,
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Finished)
		assert.True(t, executionCtx.Passed)
		assert.Equal(t, "prometheus.silence.expired", executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)
		wrappedPayload := executionCtx.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, "abc123", payload["silenceID"])
		assert.Equal(t, "expired", payload["status"])

		metadata := metadataCtx.Metadata.(ExpireSilenceNodeMetadata)
		assert.Equal(t, "abc123", metadata.SilenceID)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"silence not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"silence": "nonexistent"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to expire silence")
	})

	t.Run("execute sanitizes silence", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"silence": "  abc123  "},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)
		require.Len(t, executionCtx.Payloads, 1)
		wrappedPayload := executionCtx.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, "abc123", payload["silenceID"])
	})
}
