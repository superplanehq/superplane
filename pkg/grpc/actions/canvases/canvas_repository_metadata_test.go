package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__canvasRepositoryMetadata(t *testing.T) {
	r := support.Setup(t)

	canvas, repository := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)
	metadata := canvasRepositoryMetadataFromRepository(context.Background(), canvas, repository, r.GitProvider)

	require.NotNil(t, metadata)
	assert.Equal(t, canvas.ID.String(), metadata.CanvasId)
	assert.Equal(t, repository.RepoID, metadata.RepoId)
	assert.Equal(t, repository.Provider, metadata.Provider)
	assert.Equal(t, "main", metadata.DefaultBranch)
	assert.Contains(t, metadata.Url, "/git/")
}

func Test__canvasRepositoryMetadataForCanvas__pending(t *testing.T) {
	r := support.Setup(t)

	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		Name:           "pending-app",
	}

	metadata := canvasRepositoryMetadataForCanvas(context.Background(), canvas, r.GitProvider)
	expectedRepoID := r.GitProvider.GetRepositoryID(provider.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
		Name:           canvas.Name,
	})

	assert.Equal(t, expectedRepoID, metadata.RepoId)
	assert.Equal(t, r.GitProvider.Name(), metadata.Provider)
}
