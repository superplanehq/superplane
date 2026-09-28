package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadGitHubHostedAppConfig(t *testing.T) {
	t.Setenv(EnvGitHubAppID, "")
	t.Setenv(EnvGitHubAppSlug, "")
	t.Setenv(EnvGitHubAppPrivateKey, "")
	t.Setenv(EnvGitHubAppWebhookSecret, "")

	t.Run("empty env is not enabled", func(t *testing.T) {
		cfg := LoadGitHubHostedAppConfig()
		assert.False(t, cfg.Enabled())
	})

	t.Run("all values yield the app", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "12345")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "-----BEGIN RSA PRIVATE KEY-----\\nabc\\n-----END RSA PRIVATE KEY-----")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")

		cfg := LoadGitHubHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.Equal(t, int64(12345), cfg.ID)
		assert.Equal(t, "superplane", cfg.Slug)
		assert.Equal(t, "whsec", cfg.WebhookSecret)
		assert.Contains(t, cfg.PrivateKey, "BEGIN RSA PRIVATE KEY")
		assert.Contains(t, cfg.PrivateKey, "\n")
		assert.False(t, strings.HasPrefix(cfg.PrivateKey, `"`))
	})

	t.Run("quoted docker compose pem is usable", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "12345")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, `"-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----"`)
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")

		cfg := LoadGitHubHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.Equal(t, "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----", cfg.PrivateKey)
	})

	t.Run("invalid id is not enabled", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "nope")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "pem")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")

		assert.False(t, LoadGitHubHostedAppConfig().Enabled())
	})

	t.Run("oauth env is optional for Enabled", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "12345")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "pem")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")
		t.Setenv(EnvGitHubAppClientID, "")
		t.Setenv(EnvGitHubAppClientSecret, "")

		cfg := LoadGitHubHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.False(t, cfg.UserOAuthEnabled())
	})

	t.Run("user oauth needs client id and secret", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "12345")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "pem")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")
		t.Setenv(EnvGitHubAppClientID, "Iv1.abc")
		t.Setenv(EnvGitHubAppClientSecret, "app-secret")

		cfg := LoadGitHubHostedAppConfig()
		assert.True(t, cfg.Enabled())
		assert.True(t, cfg.UserOAuthEnabled())
		assert.Equal(t, "Iv1.abc", cfg.ClientID)
		assert.Equal(t, "app-secret", cfg.ClientSecret)
	})
}
