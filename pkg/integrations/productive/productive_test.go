package productive

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

func Test__Productive__Sync(t *testing.T) {
	p := &Productive{}

	t.Run("missing apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "",
				"organizationId": "org-1",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("missing organizationId -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "token-1",
				"organizationId": "",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "organizationId is required")
	})

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "token-1",
				"organizationId": "org-1",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "organization_memberships")
		assert.Equal(t, "token-1", httpContext.Requests[0].Header.Get(AuthTokenHeader))
		assert.Equal(t, "org-1", httpContext.Requests[0].Header.Get(OrganizationIDHeader))
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Not authorized"}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "bad-token",
				"organizationId": "org-1",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__Productive__ListResources(t *testing.T) {
	p := &Productive{}

	t.Run("unknown type returns empty", func(t *testing.T) {
		resources, err := p.ListResources("unknown", core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        &contexts.HTTPContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("projects lists connected projects", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[{"id":"1","type":"projects","attributes":{"name":"Payments"}}]}`),
			},
		}

		resources, err := p.ListResources(ResourceTypeProject, core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        httpContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "Payments", resources[0].Name)
		assert.Equal(t, ResourceTypeProject, resources[0].Type)
	})
}

// authorizedIntegration returns an integration context with valid Productive.io
// credentials configured, for tests that need a working client.
func authorizedIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken":       "token-1",
			"organizationId": "org-1",
		},
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
