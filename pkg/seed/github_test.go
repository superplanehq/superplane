package seed

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func TestSelectInstallation(t *testing.T) {
	one := common.PendingInstallation{ID: "11", AccountLogin: "acme"}
	two := common.PendingInstallation{ID: "22", AccountLogin: "other"}

	t.Run("returns a clear error when no installations exist", func(t *testing.T) {
		_, err := SelectInstallation(nil, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoGitHubInstallations)
	})

	t.Run("uses the only installation", func(t *testing.T) {
		got, err := SelectInstallation([]common.PendingInstallation{one}, "")
		require.NoError(t, err)
		assert.Equal(t, one, got)
	})

	t.Run("uses the requested installation id", func(t *testing.T) {
		got, err := SelectInstallation([]common.PendingInstallation{one, two}, "22")
		require.NoError(t, err)
		assert.Equal(t, two, got)
	})

	t.Run("lists installations when more than one exists", func(t *testing.T) {
		_, err := SelectInstallation([]common.PendingInstallation{one, two}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SUPERPLANE_SEED_GITHUB_INSTALLATION_ID")
		assert.Contains(t, err.Error(), "11")
		assert.Contains(t, err.Error(), "22")
		assert.Contains(t, err.Error(), "acme")
	})

	t.Run("rejects a missing requested installation id", func(t *testing.T) {
		_, err := SelectInstallation([]common.PendingInstallation{one}, "99")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "99")
		assert.False(t, errors.Is(err, ErrNoGitHubInstallations))
	})
}

func TestSelectRepository(t *testing.T) {
	repos := []common.Repository{{Name: "app"}, {Name: "docs"}}

	t.Run("returns a clear error when no repositories exist", func(t *testing.T) {
		_, err := SelectRepository("acme", nil, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoGitHubRepositories)
	})

	t.Run("uses the first repository as owner/name", func(t *testing.T) {
		got, err := SelectRepository("acme", repos, "")
		require.NoError(t, err)
		assert.Equal(t, "acme/app", got)
	})

	t.Run("prefixes a short requested name with the owner", func(t *testing.T) {
		got, err := SelectRepository("acme", repos, "docs")
		require.NoError(t, err)
		assert.Equal(t, "acme/docs", got)
	})

	t.Run("keeps a full requested name", func(t *testing.T) {
		got, err := SelectRepository("acme", repos, "other/repo")
		require.NoError(t, err)
		assert.Equal(t, "other/repo", got)
	})
}
