package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__HashPassword(t *testing.T) {
	t.Run("produces a verifiable hash", func(t *testing.T) {
		hash, err := HashPassword("s3cret-password")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, "s3cret-password", hash)
		assert.True(t, VerifyPassword(hash, "s3cret-password"))
	})

	t.Run("produces distinct hashes for the same password", func(t *testing.T) {
		first, err := HashPassword("same-password")
		require.NoError(t, err)
		second, err := HashPassword("same-password")
		require.NoError(t, err)
		assert.NotEqual(t, first, second, "bcrypt salts each hash")
	})
}

func Test__VerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)

	assert.True(t, VerifyPassword(hash, "correct-horse"))
	assert.False(t, VerifyPassword(hash, "wrong-password"))
	assert.False(t, VerifyPassword("not-a-valid-bcrypt-hash", "correct-horse"))
}
