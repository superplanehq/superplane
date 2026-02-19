package grafana

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__QueryDataSource__Setup(t *testing.T) {
	component := QueryDataSource{}

	t.Run("data source uid is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataSourceUid": "",
				"query":         "up",
			},
		})

		require.ErrorContains(t, err, "dataSourceUid is required")
	})

	t.Run("query is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataSourceUid": "logs",
				"query":         "",
			},
		})

		require.ErrorContains(t, err, "query is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataSourceUid": "logs",
				"query":         "{}",
			},
		})

		require.NoError(t, err)
	})
}

func Test__QueryDataSource__Configuration__UsesIntegrationResourceForDataSource(t *testing.T) {
	component := QueryDataSource{}
	fields := component.Configuration()

	var dataSourceField *configuration.Field
	for i := range fields {
		if fields[i].Name == "dataSourceUid" {
			dataSourceField = &fields[i]
			break
		}
	}

	require.NotNil(t, dataSourceField)
	require.Equal(t, configuration.FieldTypeIntegrationResource, dataSourceField.Type)
	require.NotNil(t, dataSourceField.TypeOptions)
	require.NotNil(t, dataSourceField.TypeOptions.Resource)
	require.Equal(t, resourceTypeDataSource, dataSourceField.TypeOptions.Resource.Type)
}

func Test__QueryDataSource__Execute(t *testing.T) {
	component := QueryDataSource{}

	t.Run("invalid configuration returns validation error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSourceUid": "",
				"query":         "up",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://grafana.example.com",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "dataSourceUid is required")
	})

	t.Run("successful query emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"results": {
							"A": {"frames": []}
						}
					}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSourceUid": "logs",
				"query":         "{}",
				"timeFrom":      "now-5m",
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
		assert.Equal(t, "grafana.query.result", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("request payload uses datasource uid", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"results": {}}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSourceUid": "bfcwd2pm79hj4c",
				"query":         "up",
				"timeFrom":      "now-5m",
				"timeTo":        "now",
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

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.True(t, strings.HasSuffix(request.URL.String(), "/api/ds/query"))

		body := decodeJSONBody(t, request.Body)
		queries := body["queries"].([]any)
		query := queries[0].(map[string]any)
		datasource := query["datasource"].(map[string]any)

		assert.Equal(t, "bfcwd2pm79hj4c", datasource["uid"])
		assert.Equal(t, "up", query["query"])
		assert.Equal(t, "up", query["expr"])
	})

	t.Run("defaults time range when missing", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"results": {}}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSourceUid": "logs",
				"query":         "{}",
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

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		body := decodeJSONBody(t, httpContext.Requests[0].Body)
		require.NotEmpty(t, body["from"])
		require.NotEmpty(t, body["to"])

		_, err = strconv.ParseInt(body["from"].(string), 10, 64)
		require.NoError(t, err)
		_, err = strconv.ParseInt(body["to"].(string), 10, 64)
		require.NoError(t, err)
	})

	t.Run("non-2xx response returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("bad request")),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"dataSourceUid": "logs",
				"query":         "{}",
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

		require.ErrorContains(t, err, "grafana query failed with status 400")
	})
}

func decodeJSONBody(t *testing.T, body io.ReadCloser) map[string]any {
	t.Helper()

	raw, err := io.ReadAll(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}
