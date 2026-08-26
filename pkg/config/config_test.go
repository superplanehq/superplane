package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxEmitCount(t *testing.T) {
	t.Run("defaults to 100", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "")
		assert.Equal(t, 100, MaxEmitCount())
	})

	t.Run("reads SUPERPLANE_MAX_EMIT_COUNT", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "25")
		assert.Equal(t, 25, MaxEmitCount())
	})

	t.Run("ignores invalid env values", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "not-a-number")
		assert.Equal(t, 100, MaxEmitCount())
	})
}

func TestMaxPayloadSize(t *testing.T) {
	t.Run("defaults to 512 KiB", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "")
		assert.Equal(t, 512*1024, MaxPayloadSize())
	})

	t.Run("reads SUPERPLANE_MAX_PAYLOAD_SIZE", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "8192")
		assert.Equal(t, 8192, MaxPayloadSize())
	})

	t.Run("ignores invalid env values", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "not-a-number")
		assert.Equal(t, 512*1024, MaxPayloadSize())
	})
}

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
	})

	t.Run("invalid id is not enabled", func(t *testing.T) {
		t.Setenv(EnvGitHubAppID, "nope")
		t.Setenv(EnvGitHubAppSlug, "superplane")
		t.Setenv(EnvGitHubAppPrivateKey, "pem")
		t.Setenv(EnvGitHubAppWebhookSecret, "whsec")

		assert.False(t, LoadGitHubHostedAppConfig().Enabled())
	})
}
