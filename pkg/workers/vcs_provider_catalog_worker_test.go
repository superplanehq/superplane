package workers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

type recordingVCSProviderCatalog struct {
	mu                  sync.Mutex
	installationIDs     []int64
	repositoryIDs       []int64
	blockID             int64
	release             chan struct{}
	started             chan int64
	installationError   error
	reconcilePriorities []models.VCSProviderRepositorySyncPriority
	requestRefreshes    int
}

func (c *recordingVCSProviderCatalog) ReconcileInstallRequests(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestRefreshes++
	return nil
}

func (c *recordingVCSProviderCatalog) ReconcileInstallation(
	_ context.Context,
	installationID int64,
	_ models.VCSProviderRepositorySyncPriority,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.installationIDs = append(c.installationIDs, installationID)
	return c.installationError
}

func (c *recordingVCSProviderCatalog) Reconcile(
	_ context.Context,
	priority models.VCSProviderRepositorySyncPriority,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reconcilePriorities = append(c.reconcilePriorities, priority)
	return nil
}

func (c *recordingVCSProviderCatalog) SyncRepositoryCollaborators(_ context.Context, repositoryID int64) error {
	c.mu.Lock()
	c.repositoryIDs = append(c.repositoryIDs, repositoryID)
	c.mu.Unlock()
	if c.started != nil {
		c.started <- repositoryID
	}
	if repositoryID == c.blockID {
		<-c.release
	}
	return nil
}

func (c *recordingVCSProviderCatalog) repositories() []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int64(nil), c.repositoryIDs...)
}

func (c *recordingVCSProviderCatalog) installations() []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int64(nil), c.installationIDs...)
}

func (c *recordingVCSProviderCatalog) priorities() []models.VCSProviderRepositorySyncPriority {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]models.VCSProviderRepositorySyncPriority(nil), c.reconcilePriorities...)
}

func TestVCSProviderCatalogWorkerPrioritizesRequestedReconciliation(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	require.NoError(t, models.EnqueueVCSProviderReconciliation(
		database.Conn(),
		models.ProviderGitHub,
		time.Now().Add(-time.Second),
	))

	catalog := &recordingVCSProviderCatalog{}
	worker := newTestVCSProviderCatalogWorker(catalog)
	worker.processReconcileJob(t.Context())

	assert.Equal(t, []models.VCSProviderRepositorySyncPriority{
		models.VCSProviderRepositorySyncPriorityInteractive,
	}, catalog.priorities())
}

func TestVCSProviderCatalogWorkerRefreshesInstallRequestsOnlyWhileOnePending(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	catalog := &recordingVCSProviderCatalog{}
	worker := newTestVCSProviderCatalogWorker(catalog)

	worker.refreshPendingInstallRequests(t.Context())
	assert.Zero(t, catalog.requestRefreshes)

	require.NoError(t, models.ReplaceVCSProviderInstallRequests(database.Conn(), models.ProviderGitHub, []models.VCSProviderInstallRequest{{
		RequestID:    401,
		AccountLogin: "acme",
		RequesterID:  9,
		RequestedAt:  time.Now(),
	}}))
	worker.refreshPendingInstallRequests(t.Context())
	assert.Equal(t, 1, catalog.requestRefreshes)
}

func TestVCSProviderCatalogWorkerReconcilesRequestedInstallation(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	require.NoError(t, models.EnqueueVCSProviderInstallationReconciliation(
		database.Conn(),
		models.ProviderGitHub,
		101,
		uuid.New(),
		time.Now().Add(-time.Second),
	))

	catalog := &recordingVCSProviderCatalog{}
	worker := newTestVCSProviderCatalogWorker(catalog)
	worker.processInstallationJobs(t.Context())
	worker.installationJobs.Wait()

	assert.Equal(t, []int64{101}, catalog.installations())
	var remaining int64
	require.NoError(t, database.Conn().Model(&models.VCSProviderInstallationReconcileJob{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}

func TestVCSProviderCatalogWorkerDiscardsRepeatedlyFailingInstallation(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	db := database.Conn()
	require.NoError(t, models.EnqueueVCSProviderInstallationReconciliation(
		db,
		models.ProviderGitHub,
		101,
		uuid.New(),
		time.Now().Add(-time.Second),
	))
	require.NoError(t, db.Model(&models.VCSProviderInstallationReconcileJob{}).
		Where("provider = ? AND installation_id = ?", models.ProviderGitHub, 101).
		Update("attempts", vcsProviderInstallationMaxAttempts-1).Error)

	catalog := &recordingVCSProviderCatalog{installationError: errors.New("installation is unavailable")}
	worker := newTestVCSProviderCatalogWorker(catalog)
	worker.processInstallationJobs(t.Context())
	worker.installationJobs.Wait()

	var remaining int64
	require.NoError(t, db.Model(&models.VCSProviderInstallationReconcileJob{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}

func TestVCSProviderCatalogWorkerProcessesRepositoryJobsInBatches(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	db := database.Conn()

	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
	}))
	repositories := make([]models.VCSProviderRepository, 0, vcsProviderJobBatchSize)
	repositoryIDs := make([]int64, 0, vcsProviderJobBatchSize)
	for index := range vcsProviderJobBatchSize {
		repositoryID := int64(201 + index)
		repositoryIDs = append(repositoryIDs, repositoryID)
		repositories = append(repositories, models.VCSProviderRepository{
			RepositoryID: repositoryID,
			FullName:     "acme/repository-" + string(rune('a'+index)),
		})
	}
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 101, repositories))
	for _, repositoryID := range repositoryIDs {
		require.NoError(t, models.EnqueueVCSProviderRepositorySync(
			db,
			models.ProviderGitHub,
			repositoryID,
			time.Now().Add(-time.Second),
			models.VCSProviderRepositorySyncPriorityBackground,
		))
	}

	catalog := &recordingVCSProviderCatalog{}
	worker := newTestVCSProviderCatalogWorker(catalog)
	worker.processJobs(t.Context())
	worker.repositoryJobs.Wait()

	assert.ElementsMatch(t, repositoryIDs, catalog.repositories())
	var remaining int64
	require.NoError(t, db.Model(&models.VCSProviderRepositorySyncJob{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}

func TestVCSProviderCatalogWorkerRefillsAvailableSlots(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	db := database.Conn()

	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
	}))
	repositories := make([]models.VCSProviderRepository, 0, vcsProviderJobBatchSize+1)
	for index := range vcsProviderJobBatchSize + 1 {
		repositoryID := int64(201 + index)
		repositories = append(repositories, models.VCSProviderRepository{
			RepositoryID: repositoryID,
			FullName:     "acme/repository-" + string(rune('a'+index)),
		})
	}
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 101, repositories))
	for _, repository := range repositories {
		require.NoError(t, models.EnqueueVCSProviderRepositorySync(
			db,
			models.ProviderGitHub,
			repository.RepositoryID,
			time.Now().Add(-time.Second),
			models.VCSProviderRepositorySyncPriorityInteractive,
		))
	}

	catalog := &recordingVCSProviderCatalog{
		blockID: 201,
		release: make(chan struct{}),
		started: make(chan int64, vcsProviderJobBatchSize+1),
	}
	worker := newTestVCSProviderCatalogWorker(catalog)
	worker.processJobs(t.Context())
	for range vcsProviderJobBatchSize {
		select {
		case <-catalog.started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for the first repository jobs")
		}
	}

	require.Eventually(t, func() bool {
		return len(worker.repositorySlots) == 1
	}, time.Second, 10*time.Millisecond)
	worker.processJobs(t.Context())

	select {
	case repositoryID := <-catalog.started:
		assert.Equal(t, int64(209), repositoryID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the next repository job")
	}

	close(catalog.release)
	worker.repositoryJobs.Wait()
	assert.ElementsMatch(t, []int64{201, 202, 203, 204, 205, 206, 207, 208, 209}, catalog.repositories())
}

type progressivelyVisibleVCSProviderCatalog struct {
	blockID int64
	blocked chan struct{}
	release chan struct{}
}

func (c *progressivelyVisibleVCSProviderCatalog) Reconcile(
	context.Context,
	models.VCSProviderRepositorySyncPriority,
) error {
	return nil
}

func (c *progressivelyVisibleVCSProviderCatalog) ReconcileInstallRequests(context.Context) error {
	return nil
}

func (c *progressivelyVisibleVCSProviderCatalog) ReconcileInstallation(
	context.Context,
	int64,
	models.VCSProviderRepositorySyncPriority,
) error {
	return nil
}

func (c *progressivelyVisibleVCSProviderCatalog) SyncRepositoryCollaborators(
	_ context.Context,
	repositoryID int64,
) error {
	if repositoryID == c.blockID {
		close(c.blocked)
		<-c.release
	}
	return models.ReplaceVCSProviderRepositoryCollaborators(
		database.Conn(),
		models.ProviderGitHub,
		repositoryID,
		[]models.VCSProviderRepositoryCollaborator{{ProviderUserID: 42, ProviderLogin: "octocat"}},
	)
}

func TestVCSProviderCatalogWorkerMakesRepositoriesVisibleProgressively(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	db := database.Conn()

	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 101, []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/slow"},
		{RepositoryID: 202, FullName: "acme/fast"},
	}))
	for _, repositoryID := range []int64{201, 202} {
		require.NoError(t, models.EnqueueVCSProviderRepositorySync(
			db,
			models.ProviderGitHub,
			repositoryID,
			time.Now().Add(-time.Second),
			models.VCSProviderRepositorySyncPriorityInteractive,
		))
	}

	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	catalog := &progressivelyVisibleVCSProviderCatalog{
		blockID: 201,
		blocked: make(chan struct{}),
		release: release,
	}
	worker := newTestVCSProviderCatalogWorker(catalog)
	worker.processJobs(t.Context())

	select {
	case <-catalog.blocked:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the slow repository")
	}
	require.Eventually(t, func() bool {
		repositories, err := models.ListAccessibleVCSProviderRepositories(db, models.ProviderGitHub, 42)
		return err == nil && len(repositories) == 1 && repositories[0].RepositoryID == 202
	}, time.Second, 10*time.Millisecond)

	close(release)
	worker.repositoryJobs.Wait()
	repositories, err := models.ListAccessibleVCSProviderRepositories(db, models.ProviderGitHub, 42)
	require.NoError(t, err)
	require.Len(t, repositories, 2)
}

type blockingReconcileVCSProviderCatalog struct {
	reconcileStarted       chan struct{}
	releaseReconcile       chan struct{}
	installationReconciled chan int64
	repositorySynced       chan int64
}

func (c *blockingReconcileVCSProviderCatalog) ReconcileInstallation(
	_ context.Context,
	installationID int64,
	_ models.VCSProviderRepositorySyncPriority,
) error {
	c.installationReconciled <- installationID
	return nil
}

func (c *blockingReconcileVCSProviderCatalog) ReconcileInstallRequests(context.Context) error {
	return nil
}

func (c *blockingReconcileVCSProviderCatalog) Reconcile(context.Context, models.VCSProviderRepositorySyncPriority) error {
	close(c.reconcileStarted)
	<-c.releaseReconcile
	return nil
}

func (c *blockingReconcileVCSProviderCatalog) SyncRepositoryCollaborators(_ context.Context, repositoryID int64) error {
	c.repositorySynced <- repositoryID
	return nil
}

func TestVCSProviderCatalogWorkerReconciliationDoesNotBlockRequestedWork(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	db := database.Conn()

	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 101,
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, 101, []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api"},
	}))
	require.NoError(t, models.EnqueueVCSProviderRepositorySync(
		db,
		models.ProviderGitHub,
		201,
		time.Now().Add(-time.Second),
		models.VCSProviderRepositorySyncPriorityInteractive,
	))
	require.NoError(t, models.EnqueueVCSProviderInstallationReconciliation(
		db,
		models.ProviderGitHub,
		101,
		uuid.New(),
		time.Now().Add(-time.Second),
	))

	catalog := &blockingReconcileVCSProviderCatalog{
		reconcileStarted:       make(chan struct{}),
		releaseReconcile:       make(chan struct{}),
		installationReconciled: make(chan int64, 1),
		repositorySynced:       make(chan int64, 1),
	}
	worker := newTestVCSProviderCatalogWorker(catalog)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	select {
	case <-catalog.reconcileStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconciliation")
	}
	select {
	case repositoryID := <-catalog.repositorySynced:
		assert.Equal(t, int64(201), repositoryID)
	case <-time.After(time.Second):
		t.Fatal("repository synchronization was blocked by reconciliation")
	}
	select {
	case installationID := <-catalog.installationReconciled:
		assert.Equal(t, int64(101), installationID)
	case <-time.After(time.Second):
		t.Fatal("installation reconciliation was blocked by catalog reconciliation")
	}

	cancel()
	close(catalog.releaseReconcile)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}

func TestVCSProviderInstallationRetryDelay(t *testing.T) {
	assert.Equal(t, time.Second, vcsProviderInstallationRetryDelay(1))
	assert.Equal(t, 2*time.Second, vcsProviderInstallationRetryDelay(2))
	assert.Equal(t, vcsProviderInstallationRetryMaximumDelay, vcsProviderInstallationRetryDelay(6))
	assert.Equal(t, vcsProviderInstallationRetryMaximumDelay, vcsProviderInstallationRetryDelay(100))
}

func newTestVCSProviderCatalogWorker(catalog vcsProviderCatalog) *VCSProviderCatalogWorker {
	return &VCSProviderCatalogWorker{
		provider:          models.ProviderGitHub,
		catalog:           catalog,
		logger:            log.WithField("worker", "VCSProviderCatalogWorker"),
		installationSlots: make(chan struct{}, vcsProviderInstallationJobConcurrency),
		repositorySlots:   make(chan struct{}, vcsProviderJobBatchSize),
	}
}
