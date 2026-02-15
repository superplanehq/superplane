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

func Test__CreateSilence__Setup(t *testing.T) {
	component := &CreateSilence{}

	t.Run("matchers are required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{},
			"duration":  "1h",
			"createdBy": "SuperPlane",
			"comment":   "test",
		}})
		require.ErrorContains(t, err, "at least one matcher is required")
	})

	t.Run("matcher name is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "", "value": "test", "isRegex": false, "isEqual": true}},
			"duration":  "1h",
			"createdBy": "SuperPlane",
			"comment":   "test",
		}})
		require.ErrorContains(t, err, "matcher 1: name is required")
	})

	t.Run("matcher value is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "alertname", "value": "", "isRegex": false, "isEqual": true}},
			"duration":  "1h",
			"createdBy": "SuperPlane",
			"comment":   "test",
		}})
		require.ErrorContains(t, err, "matcher 1: value is required")
	})

	t.Run("duration is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
			"duration":  "",
			"createdBy": "SuperPlane",
			"comment":   "test",
		}})
		require.ErrorContains(t, err, "duration is required")
	})

	t.Run("invalid duration returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
			"duration":  "invalid",
			"createdBy": "SuperPlane",
			"comment":   "test",
		}})
		require.ErrorContains(t, err, "invalid duration")
	})

	t.Run("createdBy is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
			"duration":  "1h",
			"createdBy": "",
			"comment":   "test",
		}})
		require.ErrorContains(t, err, "createdBy is required")
	})

	t.Run("comment is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
			"duration":  "1h",
			"createdBy": "SuperPlane",
			"comment":   "",
		}})
		require.ErrorContains(t, err, "comment is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
			"duration":  "1h",
			"createdBy": "SuperPlane",
			"comment":   "Test silence",
		}})
		require.NoError(t, err)
	})
}

func Test__CreateSilence__Execute(t *testing.T) {
	component := &CreateSilence{}

	t.Run("silence is created and emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"silenceID":"abc-123"}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Test silence",
			},
			HTTP: httpCtx,
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
		assert.Equal(t, "SuperPlane", payload["createdBy"])
		assert.Equal(t, "Test silence", payload["comment"])
		assert.NotEmpty(t, payload["startsAt"])
		assert.NotEmpty(t, payload["endsAt"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v2/silences")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"bad request"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"matchers":  []any{map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Test silence",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to create silence")
	})
}
