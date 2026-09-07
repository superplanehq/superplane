package workers

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

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
	metrics := &recordingRepositoryProvisionerMetrics{}
	worker.metrics = metrics
	require.NoError(t, worker.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerBackfill,
	))
	assertRepositoryProvisioningMetric(
		t,
		metrics,
		repositoryProvisionerTriggerBackfill,
		executorOutcomeSuccess,
		executorReasonNone,
	)

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
	require.NoError(t, worker.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerBackfill,
	))

	updated, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RepositoryStatusReady, updated.Status)
}

func Test__RepositoryProvisionerWorker_RecordsBackfillMetrics(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	metrics := &recordingRepositoryProvisionerMetrics{}
	worker := NewRepositoryProvisionerWorker("", r.GitProvider)
	worker.metrics = metrics

	worker.backfill(context.Background())

	require.Len(t, metrics.tickDurations, 1)
	assert.Positive(t, metrics.tickDurations[0])
	assert.Equal(t, []int{0}, metrics.repositoryCounts)
}

func Test__RepositoryProvisionerWorker_RecordsRepositoryCreationFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repository := createPendingRepository(t, canvas, r.GitProvider)
	provider := &failingGitProvider{
		Provider:  r.GitProvider,
		createErr: errors.New("repository service unavailable"),
	}
	metrics := &recordingRepositoryProvisionerMetrics{}
	worker := NewRepositoryProvisionerWorker("", provider)
	worker.metrics = metrics

	require.NoError(t, worker.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerEvent,
	))

	updated, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RepositoryStatusError, updated.Status)
	assertRepositoryProvisioningMetric(
		t,
		metrics,
		repositoryProvisionerTriggerEvent,
		executorOutcomeFailed,
		repositoryProvisionerReasonCreateRepository,
	)
}

func Test__RepositoryProvisionerWorker_RecordsSeedCommitFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repository := createPendingRepository(t, canvas, r.GitProvider)
	require.NoError(t, models.CreateRepositorySeedFiles(repository.ID, []models.RepositorySeedFile{
		{Path: "README.md", Content: []byte("# seeded")},
	}))

	provider := &failingGitProvider{
		Provider:  r.GitProvider,
		commitErr: errors.New("commit rejected"),
	}
	metrics := &recordingRepositoryProvisionerMetrics{}
	worker := NewRepositoryProvisionerWorker("", provider)
	worker.metrics = metrics

	require.NoError(t, worker.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerBackfill,
	))

	updated, err := models.FindRepository(canvas.OrganizationID, canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RepositoryStatusError, updated.Status)
	assertRepositoryProvisioningMetric(
		t,
		metrics,
		repositoryProvisionerTriggerBackfill,
		executorOutcomeFailed,
		repositoryProvisionerReasonCommitSeedFiles,
	)
}

func Test__RepositoryProvisionerWorker_RecordsAlreadyProcessedRepository(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repository := createPendingRepository(t, canvas, r.GitProvider)
	worker := NewRepositoryProvisionerWorker("", r.GitProvider)
	require.NoError(t, worker.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerEvent,
	))

	metrics := &recordingRepositoryProvisionerMetrics{}
	worker.metrics = metrics
	require.NoError(t, worker.provisionRepository(
		context.Background(),
		*repository,
		repositoryProvisionerTriggerBackfill,
	))

	assertRepositoryProvisioningMetric(
		t,
		metrics,
		repositoryProvisionerTriggerBackfill,
		executorOutcomeSkipped,
		executorReasonNotFound,
	)
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

type recordingRepositoryProvisionerMetrics struct {
	repositoryMetrics []repositoryProvisioningMetric
	tickDurations     []time.Duration
	repositoryCounts  []int
}

func (m *recordingRepositoryProvisionerMetrics) recordTickDuration(_ context.Context, duration time.Duration) {
	m.tickDurations = append(m.tickDurations, duration)
}

func (m *recordingRepositoryProvisionerMetrics) recordRepositoriesCount(_ context.Context, count int) {
	m.repositoryCounts = append(m.repositoryCounts, count)
}

func (m *recordingRepositoryProvisionerMetrics) recordRepository(
	_ context.Context,
	record repositoryProvisioningMetric,
) {
	m.repositoryMetrics = append(m.repositoryMetrics, record)
}

func assertRepositoryProvisioningMetric(
	t *testing.T,
	metrics *recordingRepositoryProvisionerMetrics,
	trigger, outcome, reason string,
) {
	t.Helper()
	require.Len(t, metrics.repositoryMetrics, 1)
	record := metrics.repositoryMetrics[0]
	assert.Positive(t, record.duration)
	assert.Equal(t, trigger, record.trigger)
	assert.Equal(t, outcome, record.outcome)
	assert.Equal(t, reason, record.reason)
}

func createPendingRepository(
	t *testing.T,
	canvas *models.Canvas,
	provider git.Provider,
) *models.Repository {
	t.Helper()
	repoID := provider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: canvas.OrganizationID,
		CanvasID:       canvas.ID,
	})
	repository, err := canvas.CreatePendingRepository(provider.Name(), repoID)
	require.NoError(t, err)
	return repository
}

type failingGitProvider struct {
	git.Provider
	createErr error
	commitErr error
}

func (p *failingGitProvider) CreateRepository(ctx context.Context, repoID string) (*git.Repository, error) {
	if p.createErr != nil {
		return nil, p.createErr
	}

	return p.Provider.CreateRepository(ctx, repoID)
}

func (p *failingGitProvider) Commit(ctx context.Context, repoID string, options git.CommitOptions) (string, error) {
	if p.commitErr != nil {
		return "", p.commitErr
	}

	return p.Provider.Commit(ctx, repoID, options)
}
