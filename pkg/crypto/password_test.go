package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__HashPasswordAndVerifyPassword(t *testing.T) {
	const password = "s3cr3t-p@ssw0rd"
	const wrongPassword = "wrong-password"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	assert.True(t, VerifyPassword(hash, password))
	assert.False(t, VerifyPassword(hash, wrongPassword))
}
