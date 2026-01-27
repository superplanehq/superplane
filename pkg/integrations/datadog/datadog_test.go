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
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"site":   "",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "site is required")
	})

	t.Run("no apiKey -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("no appKey -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.eu",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
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

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "invalid-key",
				"appKey": "test-app-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__Datadog__Components(t *testing.T) {
	d := &Datadog{}
	components := d.Components()

	require.Len(t, components, 1)
	assert.Equal(t, "datadog.createEvent", components[0].Name())
}

func Test__Datadog__Triggers(t *testing.T) {
	d := &Datadog{}
	triggers := d.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "datadog.onMonitorAlert", triggers[0].Name())
}

func Test__Datadog__Configuration(t *testing.T) {
	d := &Datadog{}
	config := d.Configuration()

	require.Len(t, config, 3)

	siteField := config[0]
	assert.Equal(t, "site", siteField.Name)
	assert.True(t, siteField.Required)

	apiKeyField := config[1]
	assert.Equal(t, "apiKey", apiKeyField.Name)
	assert.True(t, apiKeyField.Required)
	assert.True(t, apiKeyField.Sensitive)

	appKeyField := config[2]
	assert.Equal(t, "appKey", appKeyField.Name)
	assert.True(t, appKeyField.Required)
	assert.True(t, appKeyField.Sensitive)
}

func Test__Datadog__InstallationInstructions(t *testing.T) {
	d := &Datadog{}
	instructions := d.InstallationInstructions()

	assert.NotEmpty(t, instructions)
	assert.Contains(t, instructions, "X-Superplane-Signature-256")
	assert.Contains(t, instructions, "webhook")
	assert.Contains(t, instructions, "HMAC-SHA256")
}
