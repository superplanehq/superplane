package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
)

func Test__HostedAppFromEnv(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "")
	t.Setenv(config.EnvSentryAppClientID, "")
	t.Setenv(config.EnvSentryAppClientSecret, "")

	t.Run("empty env is not configured", func(t *testing.T) {
		_, ok := HostedAppFromEnv()
		assert.False(t, ok)
		assert.False(t, HostedAppConfigured())
	})

	t.Run("all values yield the app", func(t *testing.T) {
		t.Setenv(config.EnvSentryAppSlug, "superplane")
		t.Setenv(config.EnvSentryAppClientID, "cid")
		t.Setenv(config.EnvSentryAppClientSecret, "csecret")

		app, ok := HostedAppFromEnv()
		require.True(t, ok)
		assert.Equal(t, "superplane", app.Slug)
		assert.Equal(t, "cid", app.ClientID)
		assert.Equal(t, "csecret", app.ClientSecret)
	})
}

func Test__PreferHostedInstall(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	assert.True(t, PreferHostedInstall("sentry", nil))
	assert.True(t, PreferHostedInstall("sentry", map[string]any{}))
	assert.False(t, PreferHostedInstall("sentry", map[string]any{"privateApp": true}))
	assert.False(t, PreferHostedInstall("github", nil))
}

func Test__HostedAppURLs(t *testing.T) {
	assert.Equal(
		t,
		"https://sentry.io/sentry-apps/superplane/external-install/",
		HostedAppExternalInstallURL("superplane"),
	)
	assert.Equal(
		t,
		"https://app.example/api/v1/sentry/app/install?state=abc+def",
		HostedAppInstallURL("https://app.example", "abc def"),
	)
	assert.Equal(t, "https://app.example/api/v1/sentry/app/setup", HostedAppSetupURL("https://app.example"))
	assert.Equal(t, "https://hooks.example/api/v1/sentry/app/webhook", HostedAppWebhookURL("https://hooks.example"))
}
