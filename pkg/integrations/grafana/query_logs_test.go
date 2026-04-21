package grafana

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

func Test__QueryLogs__Execute(t *testing.T) {
	component := QueryLogs{}

	t.Run("request payload uses loki query shape", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": 2,
						"uid": "loki-uid",
						"name": "Loki",
						"type": "loki",
						"url": "http://loki:3100",
						"isDefault": false
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"results": {"A": {"frames": []}}}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSource": "loki-uid",
				"query":      `{job="api"} |= "error"`,
				"timeFrom":   "now-15m",
				"timeTo":     "now",
				"limit":      25,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://grafana.example.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "grafana.logs.result", execCtx.Type)
		require.Len(t, httpContext.Requests, 2)

		request := httpContext.Requests[1]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.True(t, strings.HasSuffix(request.URL.String(), "/api/ds/query"))

		body := decodeJSONBody(t, request.Body)
		assert.Equal(t, "now-15m", body["from"])
		assert.Equal(t, "now", body["to"])

		queries := body["queries"].([]any)
		query := queries[0].(map[string]any)
		datasource := query["datasource"].(map[string]any)

		assert.Equal(t, "loki-uid", datasource["uid"])
		assert.Equal(t, `{job="api"} |= "error"`, query["query"])
		assert.Equal(t, `{job="api"} |= "error"`, query["expr"])
		assert.Equal(t, float64(25), query["maxLines"])
	})

	t.Run("evaluates expression time range", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": 2,
						"uid": "loki-uid",
						"name": "Loki",
						"type": "loki",
						"url": "http://loki:3100",
						"isDefault": false
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"results": {"A": {"frames": []}}}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSource": "loki-uid",
				"query":      `{job="api"} |= "error"`,
				"timeFrom":   `{{ now() - duration("15m") }}`,
				"timeTo":     `{{ now() }}`,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://grafana.example.com",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Expressions: testExpressionContext{
				run: func(expression string) (any, error) {
					switch expression {
					case `now() - duration("15m")`:
						return time.Date(2026, 4, 9, 7, 45, 0, 0, time.UTC), nil
					case `now()`:
						return time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC), nil
					default:
						t.Fatalf("unexpected expression: %s", expression)
						return nil, nil
					}
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)

		body := decodeJSONBody(t, httpContext.Requests[1].Body)
		assert.Equal(t, "1775720700000", body["from"])
		assert.Equal(t, "1775721600000", body["to"])
	})

	t.Run("non-loki data source returns validation error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": 1,
						"uid": "grafanacloud-prom",
						"name": "Prometheus",
						"type": "prometheus",
						"url": "http://prometheus:9090",
						"isDefault": true
					}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSource": "grafanacloud-prom",
				"query":      `{job="api"} |= "error"`,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://grafana.example.com",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, `must be a Loki data source`)
		require.Len(t, httpContext.Requests, 1)
	})
}
