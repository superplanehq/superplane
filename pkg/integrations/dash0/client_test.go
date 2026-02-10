package dash0

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	t.Run("missing apiToken -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "https://api.us-west-2.aws.dash0.com",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, integrationCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "api token")
	})

	t.Run("missing baseURL -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, integrationCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "baseURL is required")
	})

	t.Run("successful client creation", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "token123", client.Token)
		assert.Equal(t, "https://api.us-west-2.aws.dash0.com", client.BaseURL)
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com", client.LogsIngestURL)
		assert.Equal(t, "default", client.Dataset)
	})

	t.Run("baseURL with /api/prometheus suffix -> strips suffix", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com/api/prometheus",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "https://api.us-west-2.aws.dash0.com", client.BaseURL)
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com", client.LogsIngestURL)
		assert.Equal(t, "default", client.Dataset)
	})

	t.Run("custom dataset -> uses provided value", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
				"dataset":  "production",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, integrationCtx)

		require.NoError(t, err)
		assert.Equal(t, "production", client.Dataset)
	})
}

func Test__deriveLogsIngestURL(t *testing.T) {
	t.Run("converts api host to ingress host", func(t *testing.T) {
		actual := deriveLogsIngestURL("https://api.us-west-2.aws.dash0.com")
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com", actual)
	})

	t.Run("keeps non-api hosts unchanged", func(t *testing.T) {
		actual := deriveLogsIngestURL("https://ingest.internal.example.com")
		assert.Equal(t, "https://ingest.internal.example.com", actual)
	})

	t.Run("removes path before building ingest URL", func(t *testing.T) {
		actual := deriveLogsIngestURL("https://api.us-west-2.aws.dash0.com/api/prometheus")
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com", actual)
	})
}

func Test__Client__ExecutePrometheusInstantQuery(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": [
									{
										"metric": {"service_name": "test"},
										"value": [1234567890, "1"]
									}
								]
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		response, err := client.ExecutePrometheusInstantQuery("up", "default")
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.Equal(t, "Bearer token123", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("query failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","errorType":"bad_data","error":"parse error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		_, err = client.ExecutePrometheusInstantQuery("invalid query", "default")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("response too large -> error", func(t *testing.T) {
		// Create a response larger than MaxResponseSize (1MB)
		largeBody := strings.Repeat("x", MaxResponseSize+1)
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(largeBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		_, err = client.ExecutePrometheusInstantQuery("up", "default")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "response too large")
	})
}

func Test__Client__ExecutePrometheusRangeQuery(t *testing.T) {
	t.Run("successful range query", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "matrix",
								"result": [
									{
										"metric": {"service_name": "test"},
										"values": [[1234567890, "1"], [1234567900, "2"]]
									}
								]
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		response, err := client.ExecutePrometheusRangeQuery("up", "default", "now-5m", "now", "15s")
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query_range")
	})
}

func Test__Client__GetCheckDetails(t *testing.T) {
	t.Run("uses fallback endpoint on not found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"check-123","name":"Checkout latency"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		response, err := client.GetCheckDetails("check-123", false)
		require.NoError(t, err)
		assert.Equal(t, "check-123", response["checkId"])
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/failed-checks/check-123")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/api/alerting/check-rules/check-123")
		assert.Equal(t, "default", httpContext.Requests[0].URL.Query().Get("dataset"))
		assert.Equal(t, "default", httpContext.Requests[1].URL.Query().Get("dataset"))
	})
}

func Test__Client__ListSyntheticChecks(t *testing.T) {
	t.Run("parses synthetic checks from string ids", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`["check-a","check-b"]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		checks, err := client.ListSyntheticChecks()
		require.NoError(t, err)
		require.Len(t, checks, 2)
		assert.Equal(t, "check-a", checks[0].ID)
		assert.Equal(t, "check-a", checks[0].Name)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "default", httpContext.Requests[0].URL.Query().Get("dataset"))
	})
}

func Test__Client__SendLogEvents(t *testing.T) {
	t.Run("sends otlp logs request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		request := OTLPLogsRequest{
			ResourceLogs: []OTLPResourceLogs{
				{
					Resource: OTLPResource{
						Attributes: []OTLPKeyValue{
							{Key: "service.name", Value: otlpStringValue("superplane.workflow")},
						},
					},
					ScopeLogs: []OTLPScopeLogs{
						{
							Scope: OTLPScope{Name: "superplane.workflow"},
							LogRecords: []OTLPLogRecord{
								{
									TimeUnixNano:   "1739102400000000000",
									SeverityText:   "INFO",
									SeverityNumber: 9,
									Body:           otlpStringValue("workflow deployed"),
								},
							},
						},
					},
				},
			},
		}

		response, err := client.SendLogEvents(request)
		require.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/v1/logs")
		assert.Contains(t, httpContext.Requests[0].URL.Host, "ingress.")
		assert.Equal(t, "Bearer token123", httpContext.Requests[0].Header.Get("Authorization"))
	})
}
