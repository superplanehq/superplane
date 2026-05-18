package cloudflare

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Cloudflare__Sync(t *testing.T) {
	c := &Cloudflare{}

	t.Run("no apiToken -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
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

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":  "token123",
				"accountId": "acc123",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones", httpContext.Requests[0].URL.String())

		metadata := integrationCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Zones, 1)
		assert.Equal(t, "zone123", metadata.Zones[0].ID)
		assert.Equal(t, "example.com", metadata.Zones[0].Name)
		assert.Equal(t, "acc123", metadata.AccountID)
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

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":  "invalid-token",
				"accountId": "account123",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones", httpContext.Requests[0].URL.String())
		assert.Nil(t, integrationCtx.Metadata)
	})
}

func Test__Cloudflare__Configuration(t *testing.T) {
	c := &Cloudflare{}
	fields := c.Configuration()

	require.Len(t, fields, 2)
	assert.Equal(t, "apiToken", fields[0].Name)
	assert.True(t, fields[0].Required)
	assert.Equal(t, "accountId", fields[1].Name)
	assert.False(t, fields[1].Required)
}

func Test__Cloudflare__ListResources(t *testing.T) {
	c := &Cloudflare{}

	t.Run("list zones from metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": [
							{"id": "zone1", "name": "example.com", "status": "active"},
							{"id": "zone2", "name": "test.com", "status": "active"}
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

		resources, err := c.ListResources("zone", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "zone", resources[0].Type)
		assert.Equal(t, "example.com", resources[0].Name)
		assert.Equal(t, "zone1", resources[0].ID)
	})

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := c.ListResources("unknown", core.ListResourcesContext{
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

	t.Run("certificate pack list logs zone errors and continues", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"message":"temporary failure"}]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": [{"id": "pack-123", "hosts": ["app.example.com"]}]
					}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token123"},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "zone-failing", Name: "failing.example.com"},
					{ID: "zone-working", Name: "working.example.com"},
				},
			},
		}
		var logs bytes.Buffer
		logger := logrus.New()
		logger.SetOutput(&logs)
		logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

		resources, err := c.ListResources("certificate_pack", core.ListResourcesContext{
			Logger:      logrus.NewEntry(logger),
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "certificate_pack", resources[0].Type)
		assert.Equal(t, "working.example.com - app.example.com", resources[0].Name)
		assert.Equal(t, "zone-working/pack-123", resources[0].ID)
		assert.Contains(t, logs.String(), "failed to list certificate packs for zone, skipping")
		assert.Contains(t, logs.String(), "zone-failing")
		assert.Contains(t, logs.String(), "failing.example.com")
	})
}
