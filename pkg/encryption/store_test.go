package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestStoreCreatesDataKeysAndRewrapsMaster(t *testing.T) {
	ctx := t.Context()
	db := database.DB(context.Background())
	require.NoError(t, database.VerifyTestDatabase(db))
	clean := func() {
		require.NoError(t, db.Exec("DELETE FROM application_encryption_migrations").Error)
		require.NoError(t, db.Exec("DELETE FROM application_encryption_keys").Error)
	}
	clean()
	t.Cleanup(clean)

	v1 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	v2 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	masters, err := crypto.ParseLocalMasterKeys("local-v1:" + v1 + ",local-v2:" + v2)
	require.NoError(t, err)
	store := NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	encryptor := crypto.NewVersionedEncryptor(store, nil)
	oldValue, err := encryptor.Encrypt(ctx, []byte("old value"), []byte("record"))
	require.NoError(t, err)

	store = NewStore(masters, "local-v2")
	require.NoError(t, store.Initialize(ctx))
	encryptor = crypto.NewVersionedEncryptor(store, nil)
	newID, err := store.CreateAndActivateDataKey(ctx)
	require.NoError(t, err)
	newValue, err := encryptor.Encrypt(ctx, []byte("new value"), []byte("record"))
	require.NoError(t, err)
	keys, activeID, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.Equal(t, newID, activeID)
	require.Equal(t, models.ApplicationEncryptionKeyActive, keys[1].State)
	newKey, err := models.LoadApplicationEncryptionKey(db, newID)
	require.NoError(t, err)
	require.Equal(t, "local-v2", newKey.MasterKeyRef)
	require.NoError(t, store.Rewrap(ctx))

	for _, item := range []struct {
		ciphertext []byte
		plain      string
	}{{oldValue, "old value"}, {newValue, "new value"}} {
		decrypted, err := encryptor.Decrypt(ctx, item.ciphertext, []byte("record"))
		require.NoError(t, err)
		require.Equal(t, item.plain, string(decrypted))
	}

	v2Only, err := crypto.ParseLocalMasterKeys("local-v2:" + v2)
	require.NoError(t, err)
	require.NoError(t, NewStore(v2Only, "local-v2").Initialize(ctx))
	for _, key := range keys {
		stored, err := models.LoadApplicationEncryptionKey(db, key.ID)
		require.NoError(t, err)
		require.Equal(t, "local-v2", stored.MasterKeyRef)
	}
}
