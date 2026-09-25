package github

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func TestFilterWritableRepositories(t *testing.T) {
	repositories := []common.Repository{
		{ID: 1, Name: "api"},
		{ID: 2, Name: "web"},
		{ID: 3, Name: "docs"},
	}
	permissions := map[string]string{"api": "admin", "web": "write", "docs": "read"}

	writable, err := filterWritableRepositories(
		context.Background(),
		"acme",
		"member",
		repositories,
		func(_ context.Context, owner, repository, username string) (*gh.RepositoryPermissionLevel, error) {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "member", username)
			return &gh.RepositoryPermissionLevel{Permission: gh.Ptr(permissions[repository])}, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []common.Repository{
		{ID: 1, Name: "acme/api"},
		{ID: 2, Name: "acme/web"},
	}, writable)
}

func TestFilterWritableRepositoriesFailsClosed(t *testing.T) {
	repositories := []common.Repository{{ID: 1, Name: "api"}}

	writable, err := filterWritableRepositories(
		context.Background(),
		"acme",
		"member",
		repositories,
		func(context.Context, string, string, string) (*gh.RepositoryPermissionLevel, error) {
			return nil, errors.New("GitHub unavailable")
		},
	)

	assert.Error(t, err)
	assert.Empty(t, writable)
}

func TestHasRepositoryWritePermission(t *testing.T) {
	assert.True(t, hasRepositoryWritePermission("admin"))
	assert.True(t, hasRepositoryWritePermission("write"))
	assert.False(t, hasRepositoryWritePermission("read"))
	assert.False(t, hasRepositoryWritePermission("none"))
}

func TestRetainInstalledRepositoriesDoesNotGrantNewRepositories(t *testing.T) {
	granted := []common.Repository{{ID: 1, Name: "acme/api"}, {ID: 2, Name: "acme/web"}}
	installed := []common.Repository{{ID: 2, Name: "web"}, {ID: 3, Name: "new"}}

	assert.Equal(t, []common.Repository{{ID: 2, Name: "acme/web"}}, retainInstalledRepositories(granted, installed))
}
