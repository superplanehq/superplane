package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestApplicationEncryptionRequiresMasterKeysAndWritesVersionedValues(t *testing.T) {
	ctx := t.Context()
	db := database.DB(context.Background())
	require.NoError(t, database.VerifyTestDatabase(db))
	clean := func() {
		require.NoError(t, db.Exec("DELETE FROM application_encryption_migrations").Error)
		require.NoError(t, db.Exec("DELETE FROM application_encryption_keys").Error)
	}
	clean()
	t.Cleanup(clean)

	t.Setenv("ENCRYPTION_MASTER_KEYS", "")
	_, err := loadApplicationEncryption(ctx)
	require.ErrorContains(t, err, "ENCRYPTION_MASTER_KEYS is required")

	t.Setenv("ENCRYPTION_MASTER_KEYS", "env://server-test-v1:"+base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	configuration, err := loadApplicationEncryption(ctx)
	require.NoError(t, err)
	require.NotNil(t, configuration.keyStore)
	value, err := configuration.encryptor.Encrypt(ctx, []byte("secret"), []byte("record"))
	require.NoError(t, err)
	_, versioned, err := crypto.EncryptedDataKeyID(value)
	require.NoError(t, err)
	require.True(t, versioned)
	plain, err := configuration.encryptor.Decrypt(ctx, value, []byte("record"))
	require.NoError(t, err)
	require.Equal(t, "secret", string(plain))
}
