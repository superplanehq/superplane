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
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"silenceID": ""}})
		require.ErrorContains(t, err, "silenceID is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"silenceID": "abc-123"}})
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

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"silenceID": "abc-123"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Finished)
		assert.True(t, executionCtx.Passed)
		assert.Equal(t, PrometheusSilencePayloadType, executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)

		payload := executionCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "abc-123", payload["silenceID"])
		assert.Equal(t, "expired", payload["state"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v2/silence/abc-123")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
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
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to expire silence")
	})
}
