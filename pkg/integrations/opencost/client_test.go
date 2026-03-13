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

func Test__NewClient(t *testing.T) {
	httpCtx := &contexts.HTTPContext{}

	t.Run("missing baseURL returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"authType": AuthTypeNone}}
		_, err := NewClient(httpCtx, integrationCtx)
		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("invalid auth type returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": "invalid",
		}}
		_, err := NewClient(httpCtx, integrationCtx)
		require.ErrorContains(t, err, "invalid authType")
	})

	t.Run("basic auth requires username and password", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeBasic,
		}}
		_, err := NewClient(httpCtx, integrationCtx)
		require.ErrorContains(t, err, "username is required")
	})

	t.Run("creates bearer client", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":     "http://opencost.example.com:9003/",
			"authType":    AuthTypeBearer,
			"bearerToken": "secret-token",
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)
		assert.Equal(t, "http://opencost.example.com:9003", client.baseURL)
		assert.Equal(t, AuthTypeBearer, client.authType)
	})

	t.Run("creates no-auth client", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)
		assert.Equal(t, AuthTypeNone, client.authType)
	})
}

func Test__Client__GetAllocation(t *testing.T) {
	t.Run("successful allocation query", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [
							{
								"production": {
									"name": "production",
									"properties": {"namespace": "production"},
									"window": {"start": "2026-02-21T00:00:00Z", "end": "2026-02-22T00:00:00Z"},
									"start": "2026-02-21T00:00:00Z",
									"end": "2026-02-22T00:00:00Z",
									"cpuCost": 45.23,
									"gpuCost": 0,
									"ramCost": 38.92,
									"pvCost": 8.75,
									"networkCost": 12.5,
									"totalCost": 105.4,
									"cpuEfficiency": 0.42,
									"ramEfficiency": 0.61,
									"totalEfficiency": 0.51
								}
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		data, err := client.GetAllocation("1d", "namespace")
		require.NoError(t, err)
		require.Len(t, data, 1)
		require.Contains(t, data[0], "production")
		assert.Equal(t, 105.4, data[0]["production"].TotalCost)

		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/allocation/compute")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "window=1d")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "aggregate=namespace")
	})

	t.Run("adds bearer auth header", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code": 200, "data": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":     "http://opencost.example.com:9003",
			"authType":    AuthTypeBearer,
			"bearerToken": "token-1",
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Bearer token-1", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("adds basic auth header", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code": 200, "data": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeBasic,
			"username": "admin",
			"password": "password",
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		username, password, ok := httpCtx.Requests[0].BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "password", password)
	})

	t.Run("non-2xx returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`unauthorized`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.ErrorContains(t, err, "status 401")
	})

	t.Run("response too large returns error", func(t *testing.T) {
		largeBody := strings.Repeat("x", MaxResponseSize+1)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(largeBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.ErrorContains(t, err, "response too large")
	})

	t.Run("invalid json returns decode error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`not-json`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.ErrorContains(t, err, "failed to decode response JSON")
	})

	t.Run("non-200 code in response returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code": 500, "message": "internal error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"baseURL":  "http://opencost.example.com:9003",
			"authType": AuthTypeNone,
		}}

		client, err := NewClient(httpCtx, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetAllocation("1d", "namespace")
		require.ErrorContains(t, err, "OpenCost API error")
	})
}
