package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__HostedAppFromEnv(t *testing.T) {
	t.Setenv(EnvGitHubAppID, "")
	t.Setenv(EnvGitHubAppSlug, "")
	t.Setenv(EnvGitHubAppPrivateKey, "")
	t.Setenv(EnvGitHubAppWebhookSecret, "")

	t.Run("empty env is not configured", func(t *testing.T) {
		_, ok := HostedAppFromEnv()
		assert.False(t, ok)
		assert.False(t, HostedAppConfigured())
	})

	t.Run("all values yield the app", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "12345")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "-----BEGIN RSA PRIVATE KEY-----\\nabc\\n-----END RSA PRIVATE KEY-----")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")

		app, ok := HostedAppFromEnv()
		require.True(t, ok)
		assert.Equal(t, int64(12345), app.ID)
		assert.Equal(t, "superplane", app.Slug)
		assert.Equal(t, "whsec", app.WebhookSecret)
		assert.Contains(t, app.PrivateKey, "BEGIN RSA PRIVATE KEY")
		assert.Contains(t, app.PrivateKey, "\n")
	})

	t.Run("invalid id is not configured", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "nope")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "pem")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")

		_, ok := HostedAppFromEnv()
		assert.False(t, ok)
	})
}

func Test__HostedAppInstallURL(t *testing.T) {
	assert.Equal(
		t,
		"https://github.com/apps/superplane/installations/new?state=abc",
		HostedAppInstallURL("superplane", "abc"),
	)
}

func Test__HostedAppAuthorizeURL(t *testing.T) {
	got := HostedAppAuthorizeURL("Iv1.abc", "https://app.example/api/v1/github/app/oauth/callback", "csrf")
	assert.Contains(t, got, "https://github.com/login/oauth/authorize?")
	assert.Contains(t, got, "client_id=Iv1.abc")
	assert.Contains(t, got, "state=csrf")
	assert.Contains(t, got, "redirect_uri=")
}

func Test__HostedAppUserOAuthEnabled(t *testing.T) {
	assert.False(t, HostedApp{ClientID: "id"}.UserOAuthEnabled())
	assert.True(t, HostedApp{ClientID: "id", ClientSecret: "secret"}.UserOAuthEnabled())
}
