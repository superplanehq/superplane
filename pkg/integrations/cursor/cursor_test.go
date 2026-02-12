package cursor

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

func Test__Cursor__Sync(t *testing.T) {
	c := &Cursor{}

	t.Run("success with cloud agent key -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-cloud-agent-key",
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
		assert.Equal(t, "https://api.cursor.com/v0/agents?limit=1", httpContext.Requests[0].URL.String())
	})

	t.Run("success with admin key -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "test-admin-key",
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
		assert.Equal(t, "https://api.cursor.com/teams/daily-usage-data", httpContext.Requests[0].URL.String())
	})

	t.Run("success with both keys -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-cloud-agent-key",
				"adminKey":       "test-admin-key",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
	})

	t.Run("no keys provided -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "one of the keys is required")
	})

	t.Run("invalid cloud agent key -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid api key"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "invalid-key",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud agent key verification failed")
	})

	t.Run("invalid admin key -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid api key"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "invalid-key",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "admin key verification failed")
	})
}

func Test__Cursor__Components(t *testing.T) {
	c := &Cursor{}
	components := c.Components()

	assert.Len(t, components, 3)

	names := make([]string, len(components))
	for i, comp := range components {
		names[i] = comp.Name()
	}

	assert.Contains(t, names, "cursor.launchAgent")
	assert.Contains(t, names, "cursor.getDailyUsageData")
	assert.Contains(t, names, "cursor.getLastMessage")
}

func Test__Cursor__ListResources(t *testing.T) {
	c := &Cursor{}

	t.Run("list models", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":["claude-3.5-sonnet","gpt-4o","o1-mini"]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-key",
			},
		}

		resources, err := c.ListResources("model", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 4)
		assert.Equal(t, "", resources[0].ID)
		assert.Equal(t, "Auto (Recommended)", resources[0].Name)
		assert.Equal(t, "claude-3.5-sonnet", resources[1].ID)
		assert.Equal(t, "gpt-4o", resources[2].ID)
		assert.Equal(t, "o1-mini", resources[3].ID)
	})

	t.Run("unknown resource type returns empty", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{}

		resources, err := c.ListResources("unknown", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
