package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
)

func TestEncryptDecryptManagementKeyUsesDistinctAAD(t *testing.T) {
	ctx := context.Background()
	encryptor := crypto.NewNoOpEncryptor()

	apiCipher, err := EncryptAPIKey(ctx, encryptor, "openrouter", "sk-or-inference")
	require.NoError(t, err)
	mgmtCipher, err := EncryptManagementKey(ctx, encryptor, "openrouter", "sk-or-mgmt")
	require.NoError(t, err)

	assert.Equal(t, []byte("hosted_llm_key:openrouter"), APIKeyAAD("openrouter"))
	assert.Equal(t, []byte("hosted_llm_mgmt_key:openrouter"), ManagementKeyAAD("openrouter"))
	assert.NotEqual(t, APIKeyAAD("openrouter"), ManagementKeyAAD("openrouter"))

	gotAPI, err := DecryptAPIKey(ctx, encryptor, "openrouter", apiCipher)
	require.NoError(t, err)
	assert.Equal(t, "sk-or-inference", gotAPI)

	gotMgmt, err := DecryptManagementKey(ctx, encryptor, "openrouter", mgmtCipher)
	require.NoError(t, err)
	assert.Equal(t, "sk-or-mgmt", gotMgmt)

	_, err = DecryptManagementKey(ctx, encryptor, "openrouter", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provisioning API key is missing")
}
