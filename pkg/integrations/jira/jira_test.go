package jira

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Jira__Sync(t *testing.T) {
	j := &Jira{}

	t.Run("instructions use setup steps", func(t *testing.T) {
		instructions := j.Instructions()
		assert.Contains(t, instructions, "**Setup steps:**")
		assert.Contains(t, instructions, "same setup box at the top of this modal")
		assert.Contains(t, instructions, "Atlassian Developer Console")
		assert.Contains(t, instructions, atlassianDeveloperConsoleURL)
		assert.Contains(t, instructions, "`read:jira-work`")
		assert.Contains(t, instructions, "`manage:jira-webhook`")
	})

	t.Run("oauth without app credentials -> browser action", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			BaseURL:       "https://superplane.example",
			Integration:   appCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Equal(t, http.MethodGet, appCtx.BrowserAction.Method)
		assert.Contains(t, appCtx.BrowserAction.URL, "https://developer.atlassian.com/console/myapps/")
		assert.Contains(t, appCtx.BrowserAction.Description, "Callback URL")
		assert.Contains(t, appCtx.BrowserAction.Description, atlassianDeveloperConsoleURL)
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("oauth without access token -> authorize browser action", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			BaseURL:       "https://superplane.example",
			Integration:   appCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)

		authURL, err := url.Parse(appCtx.BrowserAction.URL)
		require.NoError(t, err)
		assert.Equal(t, "https", authURL.Scheme)
		assert.Equal(t, "auth.atlassian.com", authURL.Host)
		assert.Equal(t, "/authorize", authURL.Path)
		assert.Equal(t, "api.atlassian.com", authURL.Query().Get("audience"))
		assert.Equal(t, "client-id", authURL.Query().Get("client_id"))
		assert.Equal(t, "https://superplane.example/api/v1/integrations/"+appCtx.ID().String()+"/callback", authURL.Query().Get("redirect_uri"))
		assert.Equal(t, "code", authURL.Query().Get("response_type"))
		assert.Equal(t, "consent", authURL.Query().Get("prompt"))

		metadata, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.State)
		assert.Equal(t, *metadata.State, authURL.Query().Get("state"))
	})

	t.Run("callback URL prefers WebhooksBaseURL when set (e.g. ngrok)", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			BaseURL:         "http://localhost:8000",
			WebhooksBaseURL: "https://example.ngrok-free.app",
			Integration:     appCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)

		authURL, err := url.Parse(appCtx.BrowserAction.URL)
		require.NoError(t, err)
		assert.Equal(t,
			"https://example.ngrok-free.app/api/v1/integrations/"+appCtx.ID().String()+"/callback",
			authURL.Query().Get("redirect_uri"),
		)
	})

	t.Run("email as client ID -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "user@example.com",
				"clientSecret": "client-secret",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			BaseURL:       "https://superplane.example",
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "not an email address")
	})

	t.Run("oauth with access token -> ready and refreshes webhooks", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `[{"id":"cloud-123","url":"https://test.atlassian.net","name":"Test Jira","scopes":["read:jira-work"]}]`),
				response(http.StatusOK, `{"accountId":"123","displayName":"Test User"}`),
				response(http.StatusOK, `[{"id":"10000","key":"TEST","name":"Test Project"}]`),
				response(http.StatusOK, `{"isLast":true,"values":[{"id":555},{"id":777}]}`),
				response(http.StatusOK, `{"expirationDate":"2030-01-01T00:00:00.000+0000"}`),
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
			CurrentSecrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			BaseURL:       "https://superplane.example",
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)

		metadata, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, AuthTypeOAuth, metadata.AuthType)
		assert.Equal(t, "cloud-123", metadata.CloudID)
		require.Len(t, metadata.Projects, 1)
		assert.Equal(t, "TEST", metadata.Projects[0].Key)

		// 4th request lists webhooks, 5th refreshes them.
		require.GreaterOrEqual(t, len(httpContext.Requests), 5)
		assert.Equal(t, http.MethodGet, httpContext.Requests[3].Method)
		assert.Contains(t, httpContext.Requests[3].URL.String(), "/rest/api/3/webhook?")
		assert.Equal(t, http.MethodPut, httpContext.Requests[4].Method)
		assert.Equal(t, "https://api.atlassian.com/ex/jira/cloud-123/rest/api/3/webhook/refresh", httpContext.Requests[4].URL.String())
	})
}
