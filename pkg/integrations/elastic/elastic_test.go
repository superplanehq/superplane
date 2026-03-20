package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Elastic__Sync(t *testing.T) {
	e := &Elastic{}

	t.Run("missing url -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "apiKey",
				"apiKey":    "test-api-key",
			},
		}

		err := e.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "url is required")
	})

	t.Run("missing kibanaUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://elastic.example.com",
				"authType": "apiKey",
				"apiKey":   "test-api-key",
			},
		}

		err := e.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "kibanaUrl is required")
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "apiKey",
				"apiKey":    "test-api-key",
			},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
				},
			},
		}

		err := e.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.ErrorContains(t, err, "invalid Elasticsearch credentials")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("invalid kibana configuration -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "apiKey",
				"apiKey":    "test-api-key",
			},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cluster_name":"test"}`)),
				},
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
				},
			},
		}

		err := e.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.ErrorContains(t, err, "invalid Kibana configuration")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("valid api key configuration -> ready", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "apiKey",
				"apiKey":    "test-api-key",
			},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cluster_name":"test"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		err := e.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "https://elastic.example.com/", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://kibana.example.com/api/actions/connectors", httpCtx.Requests[1].URL.String())
	})
}
