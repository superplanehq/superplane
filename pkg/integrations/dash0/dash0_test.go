package dash0

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

func Test__Dash0__Sync(t *testing.T) {
	d := &Dash0{}

	t.Run("no apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("no baseURL -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("successful connection test -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
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

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.Equal(t, "Bearer token123", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("connection test failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error validating connection")
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("baseURL with /api/prometheus suffix -> strips suffix", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com/api/prometheus",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		// Should not have double /api/prometheus
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.NotContains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/prometheus")
	})
}
