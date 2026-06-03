package materialize_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestOwnerFromDraftBranchName(t *testing.T) {
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("default branch name", func(t *testing.T) {
		branch := materialize.DefaultDraftBranchName(userID)
		owner := materialize.OwnerFromDraftBranchName(branch)
		require.NotNil(t, owner)
		assert.Equal(t, userID, *owner)
	})

	t.Run("suffixed branch name", func(t *testing.T) {
		branch := materialize.DefaultDraftBranchName(userID) + "-abc12345"
		owner := materialize.OwnerFromDraftBranchName(branch)
		require.NotNil(t, owner)
		assert.Equal(t, userID, *owner)
	})

	t.Run("custom branch name", func(t *testing.T) {
		owner := materialize.OwnerFromDraftBranchName("drafts/my-feature")
		assert.Nil(t, owner)
	})

	t.Run("non-draft branch", func(t *testing.T) {
		owner := materialize.OwnerFromDraftBranchName("main")
		assert.Nil(t, owner)
	})
}

func TestUniqueDraftBranchName(t *testing.T) {
	ctx := context.Background()
	gitProvider := inmemory.NewProvider()
	userID := uuid.New()
	orgID := uuid.New()
	canvasID := uuid.New()

	repoID := gitProvider.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: orgID,
		CanvasID:       canvasID,
	})
	_, err := gitProvider.CreateRepository(ctx, repoID)
	require.NoError(t, err)

	defaultName := materialize.DefaultDraftBranchName(userID)
	name, err := materialize.UniqueDraftBranchName(ctx, gitProvider, repoID, userID)
	require.NoError(t, err)
	assert.Equal(t, defaultName, name)

	require.NoError(t, gitProvider.CreateBranch(ctx, repoID, defaultName, models.CanvasGitBranchMain))

	name, err = materialize.UniqueDraftBranchName(ctx, gitProvider, repoID, userID)
	require.NoError(t, err)
	assert.NotEqual(t, defaultName, name)
	assert.Contains(t, name, defaultName+"-")
}

func TestGitBranchExists(t *testing.T) {
	ctx := context.Background()
	gitProvider := inmemory.NewProvider()
	repoID := "test-repo"
	_, err := gitProvider.CreateRepository(ctx, repoID)
	require.NoError(t, err)

	assert.True(t, materialize.GitBranchExists(ctx, gitProvider, repoID, models.CanvasGitBranchMain))
	assert.False(t, materialize.GitBranchExists(ctx, gitProvider, repoID, "drafts/missing"))
}
