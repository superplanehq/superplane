package notion

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Notion__Sync(t *testing.T) {
	n := &Notion{}

	t.Run("missing apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": ""},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"bot-1"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "secret_token"},
		}

		err := n.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/users/me")
		assert.Equal(t, "Bearer secret_token", httpContext.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, APIVersion, httpContext.Requests[0].Header.Get("Notion-Version"))
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"API token is invalid."}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "bad-token"},
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

func Test__Notion__ListResources(t *testing.T) {
	n := &Notion{}

	t.Run("unknown type returns empty", func(t *testing.T) {
		resources, err := n.ListResources("unknown", core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        &contexts.HTTPContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("databases lists databases shared with the integration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"results":[{"id":"db-1","object":"database","title":[{"plain_text":"Tasks"}]}],"has_more":false}`),
			},
		}

		resources, err := n.ListResources(ResourceTypeDatabase, core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        httpContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "db-1", resources[0].ID)
		assert.Equal(t, "Tasks", resources[0].Name)
		assert.Equal(t, ResourceTypeDatabase, resources[0].Type)
	})

	t.Run("an untitled database falls back to a readable name", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"results":[{"id":"db-1","object":"database","title":[]}],"has_more":false}`),
			},
		}

		resources, err := n.ListResources(ResourceTypeDatabase, core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        httpContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "Untitled database", resources[0].Name)
	})
}

// authorizedIntegration returns an integration context with a valid Notion
// token configured, for tests that need a working client.
func authorizedIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "secret_token"},
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// requestBody decodes the JSON body a request carried, for asserting on what a
// client asked Notion for.
func requestBody(t *testing.T, request *http.Request) map[string]any {
	t.Helper()

	require.NotNil(t, request.Body)
	raw, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	body := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}
