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
