package dokploy

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

func validIntegrationConfig() map[string]any {
	return map[string]any{
		"baseUrl":  "https://dokploy.example.com",
		"apiToken": "dokploy_test_key",
	}
}

// projectsResponse is a trimmed project.all body: Dokploy nests applications
// under environments, which are themselves nested under projects.
const projectsResponse = `[
  {
    "projectId": "proj-1",
    "name": "storefront",
    "environments": [
      {
        "environmentId": "env-1",
        "name": "production",
        "applications": [
          {"applicationId": "app-1", "name": "web", "applicationStatus": "done"},
          {"applicationId": "app-2", "name": "api", "applicationStatus": "idle"}
        ]
      }
    ]
  }
]`

func okResponse(body string) *contexts.HTTPContext {
	return &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		},
	}
}

// readMap unwraps the emitted payload envelope produced by the
// ExecutionStateContext mock.
func readMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return item
}

func Test__Dokploy__Sync(t *testing.T) {
	integration := &Dokploy{}

	t.Run("missing baseUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "k"},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "baseUrl is required")
	})

	t.Run("missing apiToken -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://dokploy.example.com"},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("baseUrl without scheme -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "dokploy.example.com",
				"apiToken": "k",
			},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "invalid baseUrl")
	})

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpCtx := okResponse(projectsResponse)
		integrationCtx := &contexts.IntegrationContext{Configuration: validIntegrationConfig()}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://dokploy.example.com/api/project.all", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "dokploy_test_key", httpCtx.Requests[0].Header.Get("x-api-key"))
	})

	t.Run("baseUrl with trailing /api is not doubled", func(t *testing.T) {
		httpCtx := okResponse(projectsResponse)
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://dokploy.example.com/api/",
				"apiToken": "dokploy_test_key",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://dokploy.example.com/api/project.all", httpCtx.Requests[0].URL.String())
	})

	t.Run("API rejects credentials -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Invalid API key"}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{Configuration: validIntegrationConfig()}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify Dokploy credentials")
		assert.Contains(t, err.Error(), "Invalid API key")
	})
}

func Test__Dokploy__ListResources(t *testing.T) {
	integration := &Dokploy{}
	integrationCtx := &contexts.IntegrationContext{Configuration: validIntegrationConfig()}

	t.Run("application resources are qualified by project and environment", func(t *testing.T) {
		httpCtx := okResponse(projectsResponse)

		resources, err := integration.ListResources(resourceTypeApplication, core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, resourceTypeApplication, resources[0].Type)
		assert.Equal(t, "storefront / production / web", resources[0].Name)
		assert.Equal(t, "app-1", resources[0].ID)
		assert.Equal(t, "storefront / production / api", resources[1].Name)
		assert.Equal(t, "app-2", resources[1].ID)
	})

	t.Run("applications without an id are skipped", func(t *testing.T) {
		httpCtx := okResponse(`[{"projectId":"p","name":"storefront","environments":[{"environmentId":"e","name":"prod","applications":[{"applicationId":"","name":"broken"},{"applicationId":"app-1","name":"ok"}]}]}]`)

		resources, err := integration.ListResources(resourceTypeApplication, core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "app-1", resources[0].ID)
	})

	t.Run("unknown resource type -> empty, no request", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		resources, err := integration.ListResources("database", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpCtx.Requests)
	})
}

func Test__Dokploy__Triggers(t *testing.T) {
	// Dokploy's webhook payload carries no event type, resource id or
	// signature, so the integration deliberately ships no triggers.
	assert.Nil(t, (&Dokploy{}).Triggers())
}

func Test__Dokploy__displayName(t *testing.T) {
	t.Run("falls back to the id when nothing is named", func(t *testing.T) {
		assert.Equal(t, "app-1", displayName(ApplicationSummary{ApplicationID: "app-1"}))
	})

	t.Run("omits blank segments", func(t *testing.T) {
		summary := ApplicationSummary{
			ApplicationID: "app-1",
			Name:          "web",
			ProjectName:   "storefront",
		}
		assert.Equal(t, "storefront / web", displayName(summary))
	})
}
