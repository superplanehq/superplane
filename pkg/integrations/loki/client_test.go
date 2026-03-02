package loki

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
	t.Run("no auth -> creates client", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)
		assert.Equal(t, "https://loki.example.com", client.baseURL)
		assert.Equal(t, AuthTypeNone, client.authType)
	})

	t.Run("basic auth without username -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeBasic,
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.ErrorContains(t, err, "username is required")
	})

	t.Run("basic auth without password -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeBasic,
				"username": "admin",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.ErrorContains(t, err, "password is required")
	})

	t.Run("bearer auth without token -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeBearer,
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.ErrorContains(t, err, "bearerToken is required")
	})

	t.Run("with tenant ID -> sets tenant ID", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
				"tenantID": "my-org",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)
		assert.Equal(t, "my-org", client.tenantID)
	})

	t.Run("normalizes trailing slashes in base URL", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com///",
				"authType": AuthTypeNone,
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)
		assert.Equal(t, "https://loki.example.com", client.baseURL)
	})
}

func Test__Client__Ready(t *testing.T) {
	t.Run("successful readiness check", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ready")),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		err := client.Ready()
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://loki.example.com/ready", httpCtx.Requests[0].URL.String())
	})

	t.Run("failed readiness check", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader("not ready")),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		err := client.Ready()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "503")
	})
}

func Test__Client__Push(t *testing.T) {
	t.Run("successful push", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		streams := []Stream{
			{
				Stream: map[string]string{"job": "test"},
				Values: [][]string{{"1708000000000000000", "hello world"}},
			},
		}

		err := client.Push(streams)
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/loki/api/v1/push")
		assert.Equal(t, "application/json", httpCtx.Requests[0].Header.Get("Content-Type"))
	})

	t.Run("push failure", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid stream"}`)),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		streams := []Stream{
			{
				Stream: map[string]string{"job": "test"},
				Values: [][]string{{"bad-ts", "hello"}},
			},
		}

		err := client.Push(streams)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func Test__Client__QueryRange(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "success",
						"data": {
							"resultType": "streams",
							"result": [
								{
									"stream": {"job": "superplane"},
									"values": [["1708000000000000000", "log line 1"]]
								}
							]
						}
					}`)),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		data, err := client.QueryRange(`{job="superplane"}`, "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z", "100")
		require.NoError(t, err)
		assert.Equal(t, "streams", data.ResultType)
		assert.NotNil(t, data.Result)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/loki/api/v1/query_range")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "query=")
	})

	t.Run("query with non-success status", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "error",
						"data": {"resultType": "", "result": []}
					}`)),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		_, err := client.QueryRange(`{invalid}`, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status")
	})

	t.Run("query with HTTP error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("parse error")),
				},
			},
		}

		client := &Client{
			baseURL:  "https://loki.example.com",
			authType: AuthTypeNone,
			http:     httpCtx,
		}

		_, err := client.QueryRange(`{invalid`, "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}
