package crypto

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__Base64String(t *testing.T) {
	t.Run("returns a decodable string of the requested size", func(t *testing.T) {
		value, err := Base64String(32)
		require.NoError(t, err)

		decoded, err := base64.URLEncoding.DecodeString(value)
		require.NoError(t, err)
		assert.Len(t, decoded, 32)
	})

	t.Run("returns different values across calls", func(t *testing.T) {
		first, err := Base64String(16)
		require.NoError(t, err)
		second, err := Base64String(16)
		require.NoError(t, err)
		assert.NotEqual(t, first, second)
	})
}

func Test__NewRandomKey(t *testing.T) {
	encryptor := NewNoOpEncryptor()

	plainKey, encrypted, err := NewRandomKey(context.Background(), encryptor, "my-key")
	require.NoError(t, err)
	assert.NotEmpty(t, plainKey)

	// NoOpEncryptor returns the plaintext unchanged.
	assert.Equal(t, []byte(plainKey), encrypted)

	decoded, err := base64.URLEncoding.DecodeString(plainKey)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)
}
