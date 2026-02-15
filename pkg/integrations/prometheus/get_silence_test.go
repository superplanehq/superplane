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

func Test__GetSilence__Setup(t *testing.T) {
	component := &GetSilence{}

	t.Run("silenceID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"silenceID": ""}})
		require.ErrorContains(t, err, "silenceID is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"silenceID": "abc-123"}})
		require.NoError(t, err)
	})
}

func Test__GetSilence__Execute(t *testing.T) {
	component := &GetSilence{}

	t.Run("silence is retrieved and emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "abc-123",
						"status": {"state": "active"},
						"matchers": [{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}],
						"startsAt": "2026-01-19T12:00:00Z",
						"endsAt": "2026-01-19T13:00:00Z",
						"createdBy": "SuperPlane",
						"comment": "Test silence"
					}`)),
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
		assert.Equal(t, "active", payload["state"])
		assert.Equal(t, "SuperPlane", payload["createdBy"])
		assert.Equal(t, "Test silence", payload["comment"])
		assert.Equal(t, "2026-01-19T12:00:00Z", payload["startsAt"])
		assert.Equal(t, "2026-01-19T13:00:00Z", payload["endsAt"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
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

		require.ErrorContains(t, err, "failed to get silence")
	})

	t.Run("sanitizes silenceID with whitespace", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "abc-123",
						"status": {"state": "expired"},
						"matchers": [],
						"startsAt": "2026-01-19T12:00:00Z",
						"endsAt": "2026-01-19T13:00:00Z",
						"createdBy": "user",
						"comment": "test"
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"silenceID": "  abc-123  "},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)
	})
}
