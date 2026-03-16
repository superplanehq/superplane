package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageImplementationsReadAndWrite(t *testing.T) {
	t.Parallel()

	storages := map[string]Storage{
		"in-memory": NewInMemoryStorage(),
		"local":     NewLocalFolderStorage(t.TempDir()),
	}

	for name, storage := range storages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, storage.Write("org-1/extensions/file.txt", strings.NewReader("hello")))

			reader, err := storage.Read("org-1/extensions/file.txt")
			require.NoError(t, err)

			content, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.Equal(t, "hello", string(content))
		})
	}
}

func TestLocalFolderStorageRejectsTraversal(t *testing.T) {
	t.Parallel()

	storage := NewLocalFolderStorage(t.TempDir())

	err := storage.Write("../escape.txt", strings.NewReader("nope"))
	require.EqualError(t, err, "storage path must stay within root dir")
}

func TestLocalFolderStorageWritesToDisk(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	storage := NewLocalFolderStorage(rootDir)

	require.NoError(t, storage.Write("nested/file.txt", strings.NewReader("content")))

	content, err := os.ReadFile(filepath.Join(rootDir, "nested", "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "content", string(content))
}
