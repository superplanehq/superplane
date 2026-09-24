package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSentryHostedAppConfig(t *testing.T) {
	t.Setenv(EnvSentryAppSlug, "")
	t.Setenv(EnvSentryAppClientID, "")
	t.Setenv(EnvSentryAppClientSecret, "")

	t.Run("empty env is not enabled", func(t *testing.T) {
		cfg := LoadSentryHostedAppConfig()
		assert.False(t, cfg.Enabled())
	})

	t.Run("all values yield the app", func(t *testing.T) {
		t.Setenv(EnvSentryAppSlug, "superplane")
		t.Setenv(EnvSentryAppClientID, "sentry-client")
		t.Setenv(EnvSentryAppClientSecret, "sentry-secret")

		cfg := LoadSentryHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.Equal(t, "superplane", cfg.Slug)
		assert.Equal(t, "sentry-client", cfg.ClientID)
		assert.Equal(t, "sentry-secret", cfg.ClientSecret)
	})

	t.Run("missing secret is not enabled", func(t *testing.T) {
		t.Setenv(EnvSentryAppSlug, "superplane")
		t.Setenv(EnvSentryAppClientID, "sentry-client")
		t.Setenv(EnvSentryAppClientSecret, "")

		assert.False(t, LoadSentryHostedAppConfig().Enabled())
	})
}
