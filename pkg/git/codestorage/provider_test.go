package codestorage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/provider"
)

func Test__Name(t *testing.T) {
	p := &Provider{}
	assert.Equal(t, provider.CodeStorageProvider, p.Name())
}

func Test__GetRepositoryID(t *testing.T) {
	p := &Provider{}
	organizationID := uuid.New()
	canvasID := uuid.New()

	id := p.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: organizationID,
		CanvasID:       canvasID,
	})

	assert.Equal(t, "orgs/"+organizationID.String()+"/canvases/"+canvasID.String(), id)
}

func Test__getPrivateKey(t *testing.T) {
	t.Run("reads the key from the inline env var", func(t *testing.T) {
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "inline-key")
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "")

		key, err := getPrivateKey()
		require.NoError(t, err)
		assert.Equal(t, []byte("inline-key"), key)
	})

	t.Run("reads the key from the file path when inline value is absent", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "key.pem")
		require.NoError(t, os.WriteFile(path, []byte("file-key"), 0o600))

		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", path)

		key, err := getPrivateKey()
		require.NoError(t, err)
		assert.Equal(t, []byte("file-key"), key)
	})

	t.Run("errors when neither the value nor the path is set", func(t *testing.T) {
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", "")

		_, err := getPrivateKey()
		require.Error(t, err)
	})

	t.Run("errors when the key file does not exist", func(t *testing.T) {
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY", "")
		t.Setenv("GIT_STORAGE_CODE_STORAGE_PRIVATE_KEY_PATH", filepath.Join(t.TempDir(), "missing.pem"))

		_, err := getPrivateKey()
		require.Error(t, err)
	})
}
