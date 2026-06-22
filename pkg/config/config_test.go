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
	t.Run("defaults to 64 KiB", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "")
		assert.Equal(t, 64*1024, MaxPayloadSize())
	})

	t.Run("reads SUPERPLANE_MAX_PAYLOAD_SIZE", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "8192")
		assert.Equal(t, 8192, MaxPayloadSize())
	})

	t.Run("ignores invalid env values", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "not-a-number")
		assert.Equal(t, 64*1024, MaxPayloadSize())
	})
}

func TestUsesDatabaseSMTPEmailSettings(t *testing.T) {
	t.Run("uses database SMTP settings when owner setup is enabled", func(t *testing.T) {
		t.Setenv("OWNER_SETUP_ENABLED", "yes")
		assert.True(t, UsesDatabaseSMTPEmailSettings())
	})

	t.Run("does not use database SMTP settings for hosted email configuration", func(t *testing.T) {
		t.Setenv("OWNER_SETUP_ENABLED", "no")
		assert.False(t, UsesDatabaseSMTPEmailSettings())
	})
}

func TestResendEmailConfigured(t *testing.T) {
	t.Run("requires all resend email settings", func(t *testing.T) {
		t.Setenv("RESEND_API_KEY", "key")
		t.Setenv("EMAIL_FROM_NAME", "SuperPlane")
		t.Setenv("EMAIL_FROM_ADDRESS", "noreply@example.com")

		assert.True(t, ResendEmailConfigured())
	})

	t.Run("returns false when a setting is missing", func(t *testing.T) {
		t.Setenv("RESEND_API_KEY", "key")
		t.Setenv("EMAIL_FROM_NAME", "SuperPlane")
		t.Setenv("EMAIL_FROM_ADDRESS", "")

		assert.False(t, ResendEmailConfigured())
	})
}
