package filesystem

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
)

func TestNewProviderUsesLocalPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BLOB_STORAGE_LOCAL_PATH", root)
	store, err := NewProvider()
	require.NoError(t, err)
	assert.Equal(t, blob.ProviderFilesystem, store.Name())
}

func TestStorePutGetHeadDelete(t *testing.T) {
	store, err := New(t.TempDir())
	require.NoError(t, err)

	ctx := context.Background()
	key := "install/orgs/abc/workspaces/def/file-1"
	payload := []byte("png-bytes")

	require.NoError(t, store.Put(ctx, key, bytes.NewReader(payload), blob.PutOptions{ContentType: "image/png"}))

	info, err := store.Head(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), info.Size)

	reader, err := store.Get(ctx, key)
	require.NoError(t, err)
	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, payload, got)

	_, err = store.SignedGetURL(ctx, key, 0)
	assert.ErrorIs(t, err, blob.ErrSignedURLUnsupported)

	require.NoError(t, store.Delete(ctx, key))
	_, err = store.Head(ctx, key)
	assert.ErrorIs(t, err, blob.ErrNotFound)
}

func TestStoreRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	require.NoError(t, err)

	_, err = store.Get(context.Background(), "../outside")
	require.Error(t, err)
	assert.NoFileExists(t, filepath.Join(filepath.Dir(root), "outside"))
}
