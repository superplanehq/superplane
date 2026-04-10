package grafana

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

func Test__QueryTraces__Execute(t *testing.T) {
	component := QueryTraces{}

	t.Run("request payload uses tempo traceql shape", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": 3,
						"uid": "tempo-uid",
						"name": "Tempo",
						"type": "tempo",
						"url": "http://tempo:3200",
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
				"dataSourceUid": "tempo-uid",
				"query":         `{ .http.status_code = 500 }`,
				"timeFrom":      "now-15m",
				"timeTo":        "now",
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
		assert.Equal(t, "grafana.traces.result", execCtx.Type)
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

		assert.Equal(t, "tempo-uid", datasource["uid"])
		assert.Equal(t, "traceql", query["queryType"])
		assert.Equal(t, `{ .http.status_code = 500 }`, query["query"])
		assert.Equal(t, float64(20), query["limit"])
		assert.Equal(t, float64(3), query["spss"])
		assert.Equal(t, "traces", query["tableType"])
		assert.Equal(t, []any{}, query["filters"])
	})

	t.Run("non-tempo data source returns validation error", func(t *testing.T) {
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
				"dataSourceUid": "grafanacloud-prom",
				"query":         `{ .http.status_code = 500 }`,
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

		require.ErrorContains(t, err, `must be a Tempo data source`)
		require.Len(t, httpContext.Requests, 1)
	})
}
