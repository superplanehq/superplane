package workers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__RepositoryProvisionerWorker_MarksRepositoryReady(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repoID := r.GitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
	})
	repository, err := canvas.CreatePendingRepository(r.GitProvider.Name(), repoID)
	require.NoError(t, err)

	worker := NewRepositoryProvisionerWorker("", r.GitProvider)
	require.NoError(t, worker.provisionRepository(context.Background(), *repository))

	updated, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RepositoryStatusReady, updated.Status)
}
