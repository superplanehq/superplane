package prometheus

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

func Test__ExpireSilence__Setup(t *testing.T) {
	component := &ExpireSilence{}

	t.Run("silenceID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"silenceID": ""},
		})
		require.ErrorContains(t, err, "silenceID is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"silenceID": "abc123"},
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
			Configuration: map[string]any{"silenceID": "abc123"},
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
			Configuration: map[string]any{"silenceID": "nonexistent"},
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

	t.Run("execute sanitizes silenceID", func(t *testing.T) {
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
			Configuration: map[string]any{"silenceID": "  abc123  "},
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
