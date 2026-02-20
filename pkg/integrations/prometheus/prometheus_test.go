package prometheus

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

func Test__Prometheus__Sync(t *testing.T) {
	integration := &Prometheus{}

	t.Run("missing baseURL returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeNone,
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("missing basic auth values returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeBasic,
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "username is required when authType is basic")
	})

	t.Run("successful sync sets ready state", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{"status":"success","data":{"resultType":"vector","result":[]}}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":            "https://prometheus.example.com",
				"authType":           AuthTypeBearer,
				"bearerToken":        "token-123",
				"webhookBearerToken": "wh-token",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1/query?query=up")
		assert.Equal(t, "Bearer token-123", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("query fails and alerts fallback succeeds", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","errorType":"bad_data","error":"parse error"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"alerts":[]}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 2)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1/query")
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/api/v1/alerts")
	})
}
