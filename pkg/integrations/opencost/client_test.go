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

func Test__Client__NewClient(t *testing.T) {
	t.Run("requires baseURL", func(t *testing.T) {
		_, err := NewClient(
			&contexts.HTTPContext{},
			&contexts.IntegrationContext{Configuration: map[string]any{"authType": AuthTypeNone}},
		)
		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("requires authType", func(t *testing.T) {
		_, err := NewClient(
			&contexts.HTTPContext{},
			&contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "http://localhost:9003"}},
		)
		require.ErrorContains(t, err, "authType is required")
	})

	t.Run("creates client with no auth", func(t *testing.T) {
		client, err := NewClient(
			&contexts.HTTPContext{},
			&contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://localhost:9003",
				"authType": AuthTypeNone,
			}},
		)

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:9003", client.baseURL)
		assert.Equal(t, AuthTypeNone, client.authType)
	})

	t.Run("creates client with basic auth", func(t *testing.T) {
		client, err := NewClient(
			&contexts.HTTPContext{},
			&contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://localhost:9003",
				"authType": AuthTypeBasic,
				"username": "admin",
				"password": "secret",
			}},
		)

		require.NoError(t, err)
		assert.Equal(t, "admin", client.username)
		assert.Equal(t, "secret", client.password)
	})

	t.Run("creates client with bearer auth", func(t *testing.T) {
		client, err := NewClient(
			&contexts.HTTPContext{},
			&contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":     "http://localhost:9003",
				"authType":    AuthTypeBearer,
				"bearerToken": "my-token",
			}},
		)

		require.NoError(t, err)
		assert.Equal(t, "my-token", client.bearerToken)
	})
}

func Test__Client__GetAllocation(t *testing.T) {
	t.Run("successful allocation request", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"default": {
								"name": "default",
								"properties": {"namespace": "default"},
								"window": {"start": "2026-01-19T00:00:00Z", "end": "2026-01-19T01:00:00Z"},
								"start": "2026-01-19T00:00:00Z",
								"end": "2026-01-19T01:00:00Z",
								"minutes": 60,
								"cpuCost": 1.5,
								"gpuCost": 0,
								"ramCost": 0.8,
								"pvCost": 0.1,
								"networkCost": 0.05,
								"totalCost": 2.45
							}
						}]
					}`)),
				},
			},
		}

		client := &Client{
			baseURL:  "http://localhost:9003",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		response, err := client.GetAllocation("1h", "namespace")
		require.NoError(t, err)
		assert.Equal(t, 200, response.Code)
		require.Len(t, response.Data, 1)

		entry, ok := response.Data[0]["default"]
		require.True(t, ok)
		assert.Equal(t, "default", entry.Name)
		assert.Equal(t, 2.45, entry.TotalCost)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("internal server error")),
				},
			},
		}

		client := &Client{
			baseURL:  "http://localhost:9003",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		_, err := client.GetAllocation("1h", "namespace")
		require.ErrorContains(t, err, "request failed with status 500")
	})
}

func Test__normalizeBaseURL(t *testing.T) {
	assert.Equal(t, "http://localhost:9003", normalizeBaseURL("http://localhost:9003/"))
	assert.Equal(t, "http://localhost:9003", normalizeBaseURL("http://localhost:9003///"))
	assert.Equal(t, "http://localhost:9003", normalizeBaseURL("http://localhost:9003"))
	assert.Equal(t, "/", normalizeBaseURL("/"))
}
