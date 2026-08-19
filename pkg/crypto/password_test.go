package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__HashPasswordAndVerifyPassword(t *testing.T) {
	password := "s3cr3t-p4ssw0rd"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	assert.True(t, VerifyPassword(hash, password))
	assert.False(t, VerifyPassword(hash, "wrong-password"))
}
