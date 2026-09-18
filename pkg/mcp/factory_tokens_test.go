package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
)

func TestUserFacingOAuthError(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "the MCP server refused the SuperPlane OAuth client", UserFacingOAuthError(nil))
	assert.Equal(t, "invalid_client", UserFacingOAuthError(errors.New("invalid_client")))
}

func TestEncryptAndDecryptResourceSecret(t *testing.T) {
	t.Parallel()
	encryptor := crypto.NewNoOpEncryptor()
	id := uuid.New()
	cipher, err := EncryptResourceSecret(context.Background(), encryptor, id, "refresh-token")
	require.NoError(t, err)
	plain, err := DecryptResourceSecret(context.Background(), encryptor, id, cipher)
	require.NoError(t, err)
	assert.Equal(t, "refresh-token", plain)
}
