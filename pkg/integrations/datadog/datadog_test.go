package datadog

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

func Test__Datadog__Sync(t *testing.T) {
	d := &Datadog{}

	t.Run("no site -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "site is required")
	})

	t.Run("no apiKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("no appKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "appKey is required")
	})

	t.Run("successful validation -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"valid": true}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.datadoghq.com/api/v1/validate")
		assert.Equal(t, "test-api-key", httpContext.Requests[0].Header.Get("DD-API-KEY"))
		assert.Equal(t, "test-app-key", httpContext.Requests[0].Header.Get("DD-APPLICATION-KEY"))
	})

	t.Run("EU site -> uses correct base URL", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"valid": true}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.eu",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.datadoghq.eu/api/v1/validate")
	})

	t.Run("validation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errors": ["Invalid API key"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "invalid-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}
