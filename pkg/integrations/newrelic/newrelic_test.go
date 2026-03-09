package newrelic

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

func Test__NewRelic__Sync(t *testing.T) {
	n := &NewRelic{}

	t.Run("no accountId -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "accountId is required")
	})

	t.Run("non-numeric accountId -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "abc123",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "accountId must be numeric")
	})

	t.Run("invalid region -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "APAC",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "region must be US or EU")
	})

	t.Run("no region -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("no userApiKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "userApiKey is required")
	})

	t.Run("no licenseKey -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "licenseKey is required")
	})

	t.Run("successful validation -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data": {"actor": {"user": {"name": "Test User"}}}}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.newrelic.com/graphql")
		assert.Equal(t, "test-user-api-key", httpContext.Requests[0].Header.Get("Api-Key"))
	})

	t.Run("EU region -> uses correct URL", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data": {"actor": {"user": {"name": "Test User"}}}}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "EU",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.eu.newrelic.com/graphql")
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errors": [{"message": "Invalid API key"}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "invalid-key",
				"licenseKey": "test-license-key",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__NewRelic__Components(t *testing.T) {
	n := &NewRelic{}
	components := n.Components()

	require.Len(t, components, 2)
	assert.Equal(t, "newrelic.reportMetric", components[0].Name())
	assert.Equal(t, "newrelic.runNRQLQuery", components[1].Name())
}

func Test__NewRelic__Triggers(t *testing.T) {
	n := &NewRelic{}
	triggers := n.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "newrelic.onIssue", triggers[0].Name())
}

func Test__NewRelic__Configuration(t *testing.T) {
	n := &NewRelic{}
	config := n.Configuration()

	require.Len(t, config, 4)

	accountIdField := config[0]
	assert.Equal(t, "accountId", accountIdField.Name)
	assert.True(t, accountIdField.Required)

	regionField := config[1]
	assert.Equal(t, "region", regionField.Name)
	assert.True(t, regionField.Required)

	userAPIKeyField := config[2]
	assert.Equal(t, "userApiKey", userAPIKeyField.Name)
	assert.True(t, userAPIKeyField.Required)
	assert.True(t, userAPIKeyField.Sensitive)

	licenseKeyField := config[3]
	assert.Equal(t, "licenseKey", licenseKeyField.Name)
	assert.True(t, licenseKeyField.Required)
	assert.True(t, licenseKeyField.Sensitive)
}
