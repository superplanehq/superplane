package github

import (
	"context"
	"errors"
	"net/http"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
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

func TestInstallationRepositoryEventRestoresSelectedRepository(t *testing.T) {
	t.Cleanup(resetBindClientHooks)
	selected := common.Repository{ID: 1, Name: "acme/api"}
	integration := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			InstallationID:       "11",
			Owner:                "acme",
			Repositories:         []common.Repository{selected},
			SelectedRepositories: []common.Repository{selected},
			RepositoryScoped:     true,
			GitHubApp:            common.GitHubAppMetadata{ID: 99},
		},
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	installed := []common.Repository{}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return installed, nil
	}
	ctx, recorder := hostedRequestContext(integration, "/api/v1/github/app/webhook", nil)
	event := &gh.InstallationRepositoriesEvent{}

	(&GitHub{}).handleInstallationRepositoriesEvent(ctx, event)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "error", integration.State)
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.Repositories)
	assert.Equal(t, []common.Repository{selected}, metadata.SelectedRepositories)

	installed = []common.Repository{{ID: 1, Name: "api"}}
	(&GitHub{}).handleInstallationRepositoriesEvent(ctx, event)

	assert.Equal(t, "ready", integration.State)
	metadata = integration.Metadata.(common.Metadata)
	assert.Equal(t, []common.Repository{selected}, metadata.Repositories)
	assert.Equal(t, []common.Repository{selected}, metadata.SelectedRepositories)
}
