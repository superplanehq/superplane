package workers

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__RepositoryProvisionerWorker_CommitsSeedFiles(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repoID := r.GitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
	})

	repository, err := canvas.CreatePendingRepository(r.GitProvider.Name(), repoID)
	require.NoError(t, err)

	require.NoError(t, models.CreateRepositorySeedFiles(repository.ID, []models.RepositorySeedFile{
		{Path: "README.md", Content: []byte("# seeded")},
		{Path: "scripts/deploy.sh", Content: []byte("#!/bin/sh\necho hi\n")},
	}))

	worker := NewRepositoryProvisionerWorker("", r.GitProvider)
	require.NoError(t, worker.provisionRepository(context.Background(), *repository))

	//
	// The repository is marked ready and the seed-file rows are cleaned up.
	//
	updated, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RepositoryStatusReady, updated.Status)

	remaining, err := models.ListRepositorySeedFiles(repository.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining)

	//
	// The seed files were committed to the canvas repository.
	//
	ctx := context.Background()
	readme := readGitFile(t, r.GitProvider, repository.RepoID, "README.md", "main")
	assert.Equal(t, "# seeded", readme)

	deploy := readGitFile(t, r.GitProvider, repository.RepoID, "scripts/deploy.sh", "main")
	assert.Equal(t, "#!/bin/sh\necho hi\n", deploy)

	files, err := r.GitProvider.ListFiles(ctx, repository.RepoID, "main")
	require.NoError(t, err)
	assert.Contains(t, files, "README.md")
	assert.Contains(t, files, "scripts/deploy.sh")
}

func Test__RepositoryProvisionerWorker_NoSeedFiles(t *testing.T) {
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

func Test__RepositoryProvisionerWorker_SeedFilesPersistedDuringInstallSurvive(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repoID := r.GitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
	})

	tx := database.Conn().Begin()
	repository, err := canvas.CreatePendingRepositoryInTransaction(tx, r.GitProvider.Name(), repoID)
	require.NoError(t, err)
	require.NoError(t, models.CreateRepositorySeedFilesInTransaction(tx, repository.ID, []models.RepositorySeedFile{
		{Path: "manifest.json", Content: []byte("{}")},
	}))
	require.NoError(t, tx.Commit().Error)

	seeded, err := models.ListRepositorySeedFiles(repository.ID)
	require.NoError(t, err)
	require.Len(t, seeded, 1)
	assert.Equal(t, "manifest.json", seeded[0].Path)
}

func readGitFile(t *testing.T, provider git.Provider, repoID, path, ref string) string {
	t.Helper()
	reader, err := provider.GetFile(context.Background(), repoID, path, ref)
	require.NoError(t, err)
	defer reader.Close()
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(body)
}
