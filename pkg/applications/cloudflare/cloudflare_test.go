package cloudflare

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

func Test__Cloudflare__Sync(t *testing.T) {
	c := &Cloudflare{}

	t.Run("no apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("api token -> successful zone list moves app to ready and sets metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": [
								{"id": "zone123", "name": "example.com", "status": "active"}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones", httpContext.Requests[0].URL.String())

		metadata := appCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Zones, 1)
		assert.Equal(t, "zone123", metadata.Zones[0].ID)
		assert.Equal(t, "example.com", metadata.Zones[0].Name)
	})

	t.Run("api token -> failed zone list returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Invalid token"}]}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "invalid-token",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones", httpContext.Requests[0].URL.String())
		assert.Nil(t, appCtx.Metadata)
	})
}

func Test__Cloudflare__ListResources(t *testing.T) {
	c := &Cloudflare{}

	t.Run("list zones from metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "zone1", Name: "example.com", Status: "active"},
					{ID: "zone2", Name: "test.com", Status: "active"},
				},
			},
		}

		resources, err := c.ListResources("zone", core.ListResourcesContext{
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "zone", resources[0].Type)
		assert.Equal(t, "example.com", resources[0].Name)
		assert.Equal(t, "zone1", resources[0].ID)
	})

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		resources, err := c.ListResources("unknown", core.ListResourcesContext{
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
