package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionSecret(t *testing.T) {
	t.Run("requires SESSION_SECRET", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "")
		_, err := SessionSecret()
		assert.Error(t, err)
	})

	t.Run("reads SESSION_SECRET", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "  secret-value  ")
		secret, err := SessionSecret()
		assert.NoError(t, err)
		assert.Equal(t, "secret-value", secret)
	})
}

func TestBaseURL(t *testing.T) {
	t.Run("trims trailing slash", func(t *testing.T) {
		t.Setenv("BASE_URL", " https://app.example.com/ ")
		assert.Equal(t, "https://app.example.com", BaseURL())
	})
}

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
