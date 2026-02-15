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

func Test__QueryRange__Setup(t *testing.T) {
	component := &QueryRange{}

	t.Run("query is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"query": "",
			"start": "2026-01-01T00:00:00Z",
			"end":   "2026-01-01T01:00:00Z",
			"step":  "15s",
		}})
		require.ErrorContains(t, err, "query is required")
	})

	t.Run("start is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"query": "up",
			"start": "",
			"end":   "2026-01-01T01:00:00Z",
			"step":  "15s",
		}})
		require.ErrorContains(t, err, "start is required")
	})

	t.Run("end is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"query": "up",
			"start": "2026-01-01T00:00:00Z",
			"end":   "",
			"step":  "15s",
		}})
		require.ErrorContains(t, err, "end is required")
	})

	t.Run("step is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"query": "up",
			"start": "2026-01-01T00:00:00Z",
			"end":   "2026-01-01T01:00:00Z",
			"step":  "",
		}})
		require.ErrorContains(t, err, "step is required")
	})

	t.Run("invalid step returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"query": "up",
			"start": "2026-01-01T00:00:00Z",
			"end":   "2026-01-01T01:00:00Z",
			"step":  "invalid",
		}})
		require.ErrorContains(t, err, "invalid step")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"query": "rate(http_requests_total[5m])",
			"start": "2026-01-01T00:00:00Z",
			"end":   "2026-01-01T01:00:00Z",
			"step":  "15s",
		}})
		require.NoError(t, err)
	})
}

func Test__QueryRange__Execute(t *testing.T) {
	component := &QueryRange{}

	t.Run("range query result is emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status":"success",
						"data":{
							"resultType":"matrix",
							"result":[{"metric":{"__name__":"up"},"values":[[1707750000,"1"],[1707750015,"1"]]}]
						}
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "up",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-01T01:00:00Z",
				"step":  "15s",
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
		assert.Equal(t, PrometheusQueryPayloadType, executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)

		payload := executionCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "up", payload["query"])
		assert.Equal(t, "matrix", payload["resultType"])
		assert.Equal(t, "2026-01-01T00:00:00Z", payload["start"])
		assert.Equal(t, "2026-01-01T01:00:00Z", payload["end"])
		assert.Equal(t, "15s", payload["step"])
		assert.NotNil(t, payload["result"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1/query_range")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "query=up")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "step=15s")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","error":"bad query"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "invalid{",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-01T01:00:00Z",
				"step":  "15s",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to execute range query")
	})
}
