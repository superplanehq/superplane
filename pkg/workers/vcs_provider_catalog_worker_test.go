package workers

import (
	"context"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

type recordingVCSProviderCatalog struct {
	mu            sync.Mutex
	repositoryIDs []int64
}

func (c *recordingVCSProviderCatalog) Reconcile(context.Context) error {
	return nil
}

func (c *recordingVCSProviderCatalog) SyncRepositoryCollaborators(_ context.Context, repositoryID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.repositoryIDs = append(c.repositoryIDs, repositoryID)
	return nil
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
		require.NoError(t, models.EnqueueVCSProviderRepositorySync(db, models.ProviderGitHub, repositoryID, time.Now().Add(-time.Second)))
	}

	catalog := &recordingVCSProviderCatalog{}
	worker := &VCSProviderCatalogWorker{
		provider: models.ProviderGitHub,
		catalog:  catalog,
		logger:   log.WithField("worker", "VCSProviderCatalogWorker"),
	}
	worker.processJobs(t.Context())

	assert.ElementsMatch(t, repositoryIDs, catalog.repositoryIDs)
	var remaining int64
	require.NoError(t, db.Model(&models.VCSProviderRepositorySyncJob{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}
