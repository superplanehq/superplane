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

func TestApplicationEncryptionRolloutWriteFormat(t *testing.T) {
	ctx := t.Context()
	db := database.DB(context.Background())
	require.NoError(t, database.VerifyTestDatabase(db))
	clean := func() {
		require.NoError(t, db.Exec("DELETE FROM application_encryption_migrations").Error)
		require.NoError(t, db.Exec("DELETE FROM application_encryption_keys").Error)
	}
	clean()
	t.Cleanup(clean)

	t.Setenv("ENCRYPTION_KEY", "1234567890abcdefghijklmnopqrstuv")
	t.Setenv("ENCRYPTION_MASTER_KEYS", "env://server-test-v1:"+base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	t.Setenv("ENCRYPTION_LEGACY_FORMAT", "aes-gcm")
	t.Setenv("ENCRYPTION_WRITE_FORMAT", "legacy")
	legacyConfig, err := loadApplicationEncryption(ctx)
	require.NoError(t, err)
	require.NotNil(t, legacyConfig.keyStore)
	require.True(t, legacyConfig.legacyWrites)
	legacyValue, err := legacyConfig.encryptor.Encrypt(ctx, []byte("legacy secret"), []byte("record"))
	require.NoError(t, err)
	_, versioned, err := crypto.EncryptedDataKeyID(legacyValue)
	require.NoError(t, err)
	require.False(t, versioned)

	t.Setenv("ENCRYPTION_WRITE_FORMAT", "versioned")
	versionedConfig, err := loadApplicationEncryption(ctx)
	require.NoError(t, err)
	require.False(t, versionedConfig.legacyWrites)
	versionedValue, err := versionedConfig.encryptor.Encrypt(ctx, []byte("versioned secret"), []byte("record"))
	require.NoError(t, err)
	_, versioned, err = crypto.EncryptedDataKeyID(versionedValue)
	require.NoError(t, err)
	require.True(t, versioned)

	for _, item := range []struct {
		value []byte
		want  string
	}{
		{legacyValue, "legacy secret"},
		{versionedValue, "versioned secret"},
	} {
		plain, err := versionedConfig.encryptor.Decrypt(ctx, item.value, []byte("record"))
		require.NoError(t, err)
		require.Equal(t, item.want, string(plain))
	}

	plain, err := legacyConfig.encryptor.Decrypt(ctx, versionedValue, []byte("record"))
	require.NoError(t, err)
	require.Equal(t, "versioned secret", string(plain))
}
