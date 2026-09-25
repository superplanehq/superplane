package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedInstallationTokenOptions(t *testing.T) {
	t.Run("keeps legacy connections installation-wide", func(t *testing.T) {
		options, err := ScopedInstallationTokenOptions(Metadata{
			Repositories: []Repository{{ID: 10, Name: "acme/api"}},
		})

		require.NoError(t, err)
		assert.Nil(t, options)
	})

	t.Run("limits new connections to selected repositories", func(t *testing.T) {
		options, err := ScopedInstallationTokenOptions(Metadata{
			RepositoryScoped: true,
			Repositories:     []Repository{{ID: 10, Name: "acme/api"}, {ID: 20, Name: "acme/web"}},
		})

		require.NoError(t, err)
		require.NotNil(t, options)
		assert.Equal(t, []int64{10, 20}, options.RepositoryIDs)
	})

	t.Run("rejects an empty repository scope", func(t *testing.T) {
		options, err := ScopedInstallationTokenOptions(Metadata{RepositoryScoped: true})

		assert.Error(t, err)
		assert.Nil(t, options)
	})

	t.Run("rejects an invalid repository ID", func(t *testing.T) {
		options, err := ScopedInstallationTokenOptions(Metadata{
			RepositoryScoped: true,
			Repositories:     []Repository{{Name: "acme/api"}},
		})

		assert.Error(t, err)
		assert.Nil(t, options)
	})
}
