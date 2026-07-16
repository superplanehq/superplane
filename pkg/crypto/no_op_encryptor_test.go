package crypto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__NoOpEncryptor(t *testing.T) {
	encryptor := NewNoOpEncryptor()
	ctx := context.Background()
	data := []byte("plaintext data")
	associatedData := []byte("context")

	encrypted, err := encryptor.Encrypt(ctx, data, associatedData)
	require.NoError(t, err)
	assert.Equal(t, data, encrypted, "NoOp encryption returns the input unchanged")

	decrypted, err := encryptor.Decrypt(ctx, encrypted, associatedData)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}
