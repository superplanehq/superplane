package jira

import (
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

func newLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logrus.NewEntry(logger)
}

func Test__Jira__Sync(t *testing.T) {
	j := &Jira{}

	t.Run("missing client credentials -> setup browser action", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "",
				"clientSecret": "",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
			Logger:        newLogger(),
			BaseURL:       "https://app.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.URL, "developer.atlassian.com")
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("client credentials present, no access token -> authorization browser action", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
			Logger:        newLogger(),
			BaseURL:       "https://app.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.URL, "auth.atlassian.com/authorize")
		assert.Contains(t, appCtx.BrowserAction.URL, "client_id=client-id")

		// State should be persisted on the integration metadata.
		md, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, md.State)
		assert.NotEmpty(t, *md.State)
	})

	t.Run("access token present -> refresh + metadata + ready", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
			Metadata: Metadata{CloudID: "cloud-123", SiteURL: "https://test.atlassian.net"},
		}
		_ = appCtx.SetSecret(OAuthAccessToken, []byte("old-token"))
		_ = appCtx.SetSecret(OAuthRefreshToken, []byte("refresh"))

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Refresh token exchange
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token":"new-token","refresh_token":"new-refresh","expires_in":3600,"token_type":"Bearer","scope":"read:jira-work"}`)),
				},
				// accessible-resources
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"cloud-123","url":"https://test.atlassian.net","name":"Test","scopes":[],"avatarUrl":""}]`)),
				},
				// GetCurrentUser
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`)),
				},
				// ListProjects
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`)),
				},
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
			Logger:        newLogger(),
			BaseURL:       "https://app.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Nil(t, appCtx.BrowserAction)
		md := appCtx.Metadata.(Metadata)
		assert.Equal(t, "cloud-123", md.CloudID)
		assert.Equal(t, "acct-1", md.AccountID)
		require.Len(t, md.Projects, 1)
		assert.Equal(t, "TEST", md.Projects[0].Key)

		// Tokens were rotated.
		secrets, _ := appCtx.GetSecrets()
		secretMap := map[string]string{}
		for _, s := range secrets {
			secretMap[s.Name] = string(s.Value)
		}
		assert.Equal(t, "new-token", secretMap[OAuthAccessToken])
		assert.Equal(t, "new-refresh", secretMap[OAuthRefreshToken])
	})

	t.Run("refresh failure clears tokens and re-prompts authorization", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		}
		_ = appCtx.SetSecret(OAuthAccessToken, []byte("old-token"))
		_ = appCtx.SetSecret(OAuthRefreshToken, []byte("expired"))

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`)),
				},
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
			Logger:        newLogger(),
			BaseURL:       "https://app.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.URL, "auth.atlassian.com/authorize")
	})
}
