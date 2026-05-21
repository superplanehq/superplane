package githubapps

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

func TestGenerateInstallationName(t *testing.T) {
	name, err := GenerateInstallationName()
	require.NoError(t, err)
	assert.Regexp(t, `^[a-z]+-[a-z]+-\d{5}$`, name)
}
