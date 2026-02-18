package opencost

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__GetAllocation(t *testing.T) {
	t.Run("successful request returns allocation data", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 28.45,
								"gpuCost": 0,
								"ramCost": 18.32,
								"pvCost": 5.67,
								"networkCost": 2.12,
								"totalCost": 54.56
							}
						}]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		data, err := client.GetAllocation("1d", "namespace")
		require.NoError(t, err)
		require.Len(t, data, 1)

		production, ok := data[0]["production"]
		require.True(t, ok)
		assert.Equal(t, "production", production.Name)
		assert.Equal(t, 54.56, production.TotalCost)
		assert.Equal(t, 28.45, production.CPUCost)
	})

	t.Run("non-200 response code returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 400,
						"message": "invalid window"
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("invalid", "namespace")
		require.ErrorContains(t, err, "returned code 400")
	})

	t.Run("HTTP error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`internal server error`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.ErrorContains(t, err, "request failed with status 500")
	})

	t.Run("basic auth sends correct headers", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{}]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeBasic,
				"username": "user",
				"password": "pass",
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		username, password, ok := httpCtx.Requests[0].BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", username)
		assert.Equal(t, "pass", password)
	})

	t.Run("bearer auth sends correct headers", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{}]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":     "http://opencost.example.com:9003",
				"authType":    AuthTypeBearer,
				"bearerToken": "my-token",
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Bearer my-token", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("request URL includes query parameters", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{}]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			},
		}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("7d", "cluster")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "window=7d")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "aggregate=cluster")
	})
}

func Test__normalizeBaseURL(t *testing.T) {
	assert.Equal(t, "http://opencost.example.com:9003", normalizeBaseURL("http://opencost.example.com:9003/"))
	assert.Equal(t, "http://opencost.example.com:9003", normalizeBaseURL("http://opencost.example.com:9003"))
	assert.Equal(t, "http://opencost.example.com:9003", normalizeBaseURL("http://opencost.example.com:9003///"))
}
