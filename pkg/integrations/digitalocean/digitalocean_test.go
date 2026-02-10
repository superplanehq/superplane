package digitalocean

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

func Test__DigitalOcean__Sync(t *testing.T) {
	d := &DigitalOcean{}

	t.Run("no apiToken -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("valid token -> fetches account, sets ready and metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"account": {
							"email": "user@example.com",
							"uuid": "abc-123",
							"status": "active",
							"droplet_limit": 25
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.digitalocean.com/v2/account", httpContext.Requests[0].URL.String())

		metadata := integrationCtx.Metadata.(Metadata)
		assert.Equal(t, "user@example.com", metadata.AccountEmail)
		assert.Equal(t, "abc-123", metadata.AccountUUID)
	})

	t.Run("invalid token (401) -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "invalid-token",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.digitalocean.com/v2/account", httpContext.Requests[0].URL.String())
		assert.Nil(t, integrationCtx.Metadata)
	})
}

func Test__DigitalOcean__ListResources(t *testing.T) {
	d := &DigitalOcean{}

	t.Run("list regions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"regions": [
							{"slug": "nyc3", "name": "New York 3", "available": true},
							{"slug": "sfo3", "name": "San Francisco 3", "available": true},
							{"slug": "old1", "name": "Old Region", "available": false}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		resources, err := d.ListResources("region", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "region", resources[0].Type)
		assert.Equal(t, "New York 3", resources[0].Name)
		assert.Equal(t, "nyc3", resources[0].ID)
	})

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := d.ListResources("unknown", core.ListResourcesContext{
			HTTP: &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
