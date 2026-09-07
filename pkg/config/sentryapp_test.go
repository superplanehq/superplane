package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSentryHostedAppConfig(t *testing.T) {
	t.Setenv(EnvSentryAppClientID, "")
	t.Setenv(EnvSentryAppClientSecret, "")
	t.Setenv(EnvSentryAppSlug, "")
	t.Setenv(EnvSentryAppBaseURL, "")

	t.Run("empty env is not enabled", func(t *testing.T) {
		cfg := LoadSentryHostedAppConfig()
		assert.False(t, cfg.Enabled())
	})

	t.Run("all values yield the app", func(t *testing.T) {
		t.Setenv(EnvSentryAppClientID, "client-id")
		t.Setenv(EnvSentryAppClientSecret, "client-secret")
		t.Setenv(EnvSentryAppSlug, "superplane")

		cfg := LoadSentryHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.Equal(t, "client-id", cfg.ClientID)
		assert.Equal(t, "client-secret", cfg.ClientSecret)
		assert.Equal(t, "superplane", cfg.Slug)
		assert.Equal(t, DefaultSentryAppBaseURL, cfg.BaseURL)
	})

	t.Run("custom base URL drops a trailing slash", func(t *testing.T) {
		t.Setenv(EnvSentryAppClientID, "client-id")
		t.Setenv(EnvSentryAppClientSecret, "client-secret")
		t.Setenv(EnvSentryAppSlug, "superplane")
		t.Setenv(EnvSentryAppBaseURL, "https://sentry.example/")

		cfg := LoadSentryHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.Equal(t, "https://sentry.example", cfg.BaseURL)
	})

	t.Run("missing slug is not enabled", func(t *testing.T) {
		t.Setenv(EnvSentryAppClientID, "client-id")
		t.Setenv(EnvSentryAppClientSecret, "client-secret")
		t.Setenv(EnvSentryAppSlug, "")

		assert.False(t, LoadSentryHostedAppConfig().Enabled())
	})
}
