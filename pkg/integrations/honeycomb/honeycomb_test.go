package honeycomb

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

func Test__Honeycomb__Sync(t *testing.T) {
	h := &Honeycomb{}

	t.Run("missing site -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"managementKey":   "keyid:secret",
				"teamSlug":        "myteam",
				"environmentSlug": "production",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "site is required")
	})

	t.Run("missing managementKey -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":            "api.honeycomb.io",
				"teamSlug":        "myteam",
				"environmentSlug": "production",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "managementKey is required")
	})

	t.Run("missing teamSlug -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":            "api.honeycomb.io",
				"managementKey":   "keyid:secret",
				"environmentSlug": "production",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "teamSlug is required")
	})

	t.Run("missing environmentSlug -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":          "api.honeycomb.io",
				"managementKey": "keyid:secret",
				"teamSlug":      "myteam",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "environmentSlug is required")
	})

	t.Run("invalid managementKey format -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":            "api.honeycomb.io",
				"managementKey":   "no-colon-here",
				"teamSlug":        "myteam",
				"environmentSlug": "production",
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "managementKey must be in format <keyID>:<secret>")
	})

	t.Run("API returns 401 -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":            "api.honeycomb.io",
				"managementKey":   "keyid:secret",
				"teamSlug":        "myteam",
				"environmentSlug": "production",
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

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.ErrorContains(t, err, "401")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("successful sync -> integration is ready, secrets are stored", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":            "api.honeycomb.io",
				"managementKey":   "keyid:secret",
				"teamSlug":        "myteam",
				"environmentSlug": "production",
			},
		}

		environmentsBody := `{
			"data": [
				{
					"id": "env-123",
					"type": "environments",
					"attributes": {"name": "Production", "slug": "production"}
				}
			]
		}`

		configKeyBody := `{
			"data": {
				"id": "cfgkey-123",
				"type": "api-keys",
				"attributes": {"secret": "cfg-secret-value"}
			}
		}`

		ingestKeyBody := `{
			"data": {
				"id": "ingestkey-id",
				"type": "api-keys",
				"attributes": {"secret": "ingest-secret-value"}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(environmentsBody)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(environmentsBody)),
				},
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(configKeyBody)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"team":{"slug":"myteam"}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(environmentsBody)),
				},
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(ingestKeyBody)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)), // <-- NEW ping response
				},
			},
		}

		err := h.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)

		assert.Len(t, httpCtx.Requests, 7)

		configurationKey, err := integrationCtx.Secrets().Get(secretNameConfigurationKey)
		require.NoError(t, err)
		assert.Equal(t, "cfg-secret-value", configurationKey)

		ingestKey, err := integrationCtx.Secrets().Get(secretNameIngestKey)
		require.NoError(t, err)
		assert.Equal(t, "ingestkey-idingest-secret-value", ingestKey)
	})
}
