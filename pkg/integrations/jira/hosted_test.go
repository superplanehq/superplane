package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appconfig "github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UseHostedOAuth(t *testing.T) {
	t.Run("false when env is empty", func(t *testing.T) {
		t.Setenv(appconfig.EnvJiraOAuthClientID, "")
		t.Setenv(appconfig.EnvJiraOAuthClientSecret, "")
		assert.False(t, UseHostedOAuth())
		assert.False(t, UseHostedInstall("jira"))
	})

	t.Run("false for other integrations", func(t *testing.T) {
		setHostedJiraOAuthEnv(t)
		assert.False(t, UseHostedInstall("github"))
	})

	t.Run("true when env is set", func(t *testing.T) {
		setHostedJiraOAuthEnv(t)
		assert.True(t, UseHostedOAuth())
		assert.True(t, UseHostedInstall("jira"))
	})
}

func Test__resolveOAuthApp(t *testing.T) {
	t.Run("prefers stored credentials", func(t *testing.T) {
		setHostedJiraOAuthEnv(t)
		app := resolveOAuthApp(&contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "stored-id",
				"clientSecret": "stored-secret",
			},
		})
		assert.Equal(t, "stored-id", app.ClientID)
		assert.Equal(t, "stored-secret", app.ClientSecret)
		assert.False(t, app.Hosted)
	})

	t.Run("uses hosted env when stored credentials are empty", func(t *testing.T) {
		setHostedJiraOAuthEnv(t)
		app := resolveOAuthApp(&contexts.IntegrationContext{})
		assert.Equal(t, "jira-client", app.ClientID)
		assert.Equal(t, "jira-secret", app.ClientSecret)
		assert.True(t, app.Hosted)
	})

	t.Run("empty when neither stored nor env credentials exist", func(t *testing.T) {
		t.Setenv(appconfig.EnvJiraOAuthClientID, "")
		t.Setenv(appconfig.EnvJiraOAuthClientSecret, "")
		app := resolveOAuthApp(&contexts.IntegrationContext{})
		assert.Equal(t, oauthApp{}, app)
	})
}

func Test__HostedOAuthCallbackURL(t *testing.T) {
	assert.Equal(t, "https://app.example/api/v1/jira/oauth/callback", HostedOAuthCallbackURL("https://app.example"))
	assert.Equal(t, "https://app.example/api/v1/jira/oauth/callback", HostedOAuthCallbackURL("https://app.example/"))
}

func Test__Sync_hostedOAuthAuthorizesWithoutStoredCredentials(t *testing.T) {
	setHostedJiraOAuthEnv(t)
	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: "11111111-1111-1111-1111-111111111111",
		Configuration: map[string]any{
			"setupReturnPath": "/org-1/workspaces/acme/lines/line-1",
		},
	}

	require.NoError(t, (&Jira{}).Sync(core.SyncContext{
		BaseURL:       "https://app.example",
		Integration:   integrationCtx,
		Configuration: integrationCtx.Configuration,
	}))

	require.NotNil(t, integrationCtx.BrowserAction)
	assert.Equal(t, "GET", integrationCtx.BrowserAction.Method)
	assert.Contains(t, integrationCtx.BrowserAction.URL, "https://auth.atlassian.com/authorize?")
	assert.Contains(t, integrationCtx.BrowserAction.URL, "client_id=jira-client")
	assert.Contains(t, integrationCtx.BrowserAction.URL, "redirect_uri=https%3A%2F%2Fapp.example%2Fapi%2Fv1%2Fjira%2Foauth%2Fcallback")
	assert.NotContains(t, integrationCtx.BrowserAction.URL, "/integrations/11111111-1111-1111-1111-111111111111/callback")

	metadata := integrationCtx.Metadata.(Metadata)
	require.NotNil(t, metadata.State)
	assert.NotEmpty(t, *metadata.State)
	assert.True(t, metadata.HostedOAuth)
	assert.Equal(t, "/org-1/workspaces/acme/lines/line-1", metadata.SetupReturnPath)
}

func Test__Sync_emptyCredentialsShowAppSetupWithoutHostedEnv(t *testing.T) {
	t.Setenv(appconfig.EnvJiraOAuthClientID, "")
	t.Setenv(appconfig.EnvJiraOAuthClientSecret, "")
	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: "11111111-1111-1111-1111-111111111111",
	}

	require.NoError(t, (&Jira{}).Sync(core.SyncContext{
		BaseURL:     "https://app.example",
		Integration: integrationCtx,
	}))

	require.NotNil(t, integrationCtx.BrowserAction)
	assert.Empty(t, integrationCtx.BrowserAction.URL)
	assert.Contains(t, integrationCtx.BrowserAction.Description, "Create an OAuth 2.0 (3LO) app")
}

func Test__callbackRedirectURL(t *testing.T) {
	integrationID := "11111111-1111-1111-1111-111111111111"
	settingsURL := "https://app.example/org-1/settings/integrations/" + integrationID

	t.Run("appends the connection id to a stored return path", func(t *testing.T) {
		got := callbackRedirectURL(core.HTTPRequestContext{
			BaseURL: "https://app.example",
			Integration: &contexts.IntegrationContext{
				IntegrationID: integrationID,
				Metadata: Metadata{
					SetupReturnPath: "/org-1/workspaces/acme/lines/line-1/setup/jira",
				},
			},
		}, settingsURL)

		assert.Equal(t,
			"https://app.example/org-1/workspaces/acme/lines/line-1/setup/jira?jiraIntegrationId="+integrationID,
			got,
		)
	})

	t.Run("falls back to settings when no return path is stored", func(t *testing.T) {
		got := callbackRedirectURL(core.HTTPRequestContext{
			BaseURL: "https://app.example",
			Integration: &contexts.IntegrationContext{
				IntegrationID: integrationID,
			},
		}, settingsURL)

		assert.Equal(t, settingsURL, got)
	})
}

func Test__isSafeIntegrationSetupReturnPath(t *testing.T) {
	assert.True(t, isSafeIntegrationSetupReturnPath("/org-1/workspaces/acme/lines/line-1"))
	assert.True(t, isSafeIntegrationSetupReturnPath("/onboarding?attempt=1"))
	assert.False(t, isSafeIntegrationSetupReturnPath("//evil.example/phishing"))
	assert.False(t, isSafeIntegrationSetupReturnPath("https://evil.example"))
	assert.False(t, isSafeIntegrationSetupReturnPath("/org-1"))
	assert.False(t, isSafeIntegrationSetupReturnPath(""))
}

func setHostedJiraOAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv(appconfig.EnvJiraOAuthClientID, "jira-client")
	t.Setenv(appconfig.EnvJiraOAuthClientSecret, "jira-secret")
}
