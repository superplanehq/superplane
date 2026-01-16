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
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"baseURL": "https://api.us-west-2.aws.dash0.com",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "api token")
	})

	t.Run("missing baseURL -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "baseURL is required")
	})

	t.Run("successful client creation", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, appCtx)

		require.NoError(t, err)
		assert.Equal(t, "token123", client.Token)
		assert.Equal(t, "https://api.us-west-2.aws.dash0.com", client.BaseURL)
	})

	t.Run("baseURL with /api/prometheus suffix -> strips suffix", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com/api/prometheus",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, appCtx)

		require.NoError(t, err)
		assert.Equal(t, "https://api.us-west-2.aws.dash0.com", client.BaseURL)
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		response, err := client.ExecutePrometheusRangeQuery("up", "default", "now-5m", "now", "15s")
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query_range")
	})
}
