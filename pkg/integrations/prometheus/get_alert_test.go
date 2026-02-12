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

func Test__GetAlert__Setup(t *testing.T) {
	component := &GetAlert{}

	t.Run("alertName is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"alertName": ""}})
		require.ErrorContains(t, err, "alertName is required")
	})

	t.Run("invalid state returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"alertName": "HighLatency", "state": "unknown"}})
		require.ErrorContains(t, err, "invalid state")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"alertName": "HighLatency", "state": AlertStateFiring}})
		require.NoError(t, err)
	})
}

func Test__GetAlert__Execute(t *testing.T) {
	component := &GetAlert{}

	t.Run("matching alert is emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{"status":"success","data":{"alerts":[
							{"state":"pending","labels":{"alertname":"OtherAlert"},"annotations":{"summary":"other"}},
							{"state":"firing","labels":{"alertname":"HighLatency","instance":"api-1"},"annotations":{"summary":"latency"},"activeAt":"2026-01-19T12:00:00Z","value":"1"}
						]}}
					`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"alertName": "HighLatency", "state": AlertStateFiring},
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
		assert.Equal(t, PrometheusAlertPayloadType, executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)
		payload := executionCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "HighLatency", payload["labels"].(map[string]string)["alertname"])
		assert.Equal(t, "firing", payload["status"])
	})

	t.Run("alert not found returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"alerts":[]}}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"alertName": "HighLatency", "state": AlertStateAny},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "was not found")
	})

	t.Run("execute sanitizes alertName and state", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{"status":"success","data":{"alerts":[
							{"state":"firing","labels":{"alertname":"HighLatency"},"annotations":{"summary":"latency"},"activeAt":"2026-01-19T12:00:00Z","value":"1"}
						]}}
					`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"alertName": "  HighLatency  ", "state": "  FIRING  "},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)
		require.Len(t, executionCtx.Payloads, 1)
		payload := executionCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "HighLatency", payload["labels"].(map[string]string)["alertname"])
		assert.Equal(t, "firing", payload["status"])
	})
}
