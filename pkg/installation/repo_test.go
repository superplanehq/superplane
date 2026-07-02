package installation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepository(t *testing.T) {
	t.Run("github.com owner repo", func(t *testing.T) {
		repo, err := ParseRepository("github.com/superplanehq/preview-env-github-digitalocean")
		require.NoError(t, err)
		assert.Equal(t, "superplanehq", repo.Owner)
		assert.Equal(t, "preview-env-github-digitalocean", repo.Name)
		assert.Equal(t, "github.com/superplanehq/preview-env-github-digitalocean", repo.String())
	})

	t.Run("https github url", func(t *testing.T) {
		repo, err := ParseRepository("https://github.com/acme/widgets")
		require.NoError(t, err)
		assert.Equal(t, "acme", repo.Owner)
		assert.Equal(t, "widgets", repo.Name)
	})

	t.Run("rejects missing owner", func(t *testing.T) {
		_, err := ParseRepository("github.com/widget")
		require.Error(t, err)
	})
}

func TestDefaultInstallationName(t *testing.T) {
	assert.Equal(t, "Preview Env Github Digitalocean", DefaultInstallationName("preview-env-github-digitalocean"))
	assert.Equal(t, "Widgets", DefaultInstallationName("widgets"))
	assert.Equal(t, "My App", DefaultInstallationName("my_app"))
	assert.Equal(t, "Acme Widgets", DefaultInstallationName("acme.widgets"))
	assert.Equal(t, "Untitled App", DefaultInstallationName(""))
}
