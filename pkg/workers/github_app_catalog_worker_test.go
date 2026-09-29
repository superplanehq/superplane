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

type recordingGitHubAppCatalog struct {
	mu            sync.Mutex
	repositoryIDs []int64
}

func (c *recordingGitHubAppCatalog) Reconcile(context.Context) error {
	return nil
}

func (c *recordingGitHubAppCatalog) SyncRepositoryCollaborators(_ context.Context, repositoryID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.repositoryIDs = append(c.repositoryIDs, repositoryID)
	return nil
}

func TestGitHubAppCatalogWorkerProcessesRepositoryJobsInBatches(t *testing.T) {
	registry := support.Setup(t)
	t.Cleanup(registry.Close)
	db := database.Conn()

	require.NoError(t, models.UpsertGitHubAppInstallation(db, &models.GitHubAppInstallation{InstallationID: 101}))
	repositories := make([]models.GitHubAppRepository, 0, githubAppJobBatchSize)
	repositoryIDs := make([]int64, 0, githubAppJobBatchSize)
	for index := range githubAppJobBatchSize {
		repositoryID := int64(201 + index)
		repositoryIDs = append(repositoryIDs, repositoryID)
		repositories = append(repositories, models.GitHubAppRepository{
			RepositoryID: repositoryID,
			FullName:     "acme/repository-" + string(rune('a'+index)),
		})
	}
	require.NoError(t, models.ReplaceGitHubAppRepositories(db, 101, repositories))
	for _, repositoryID := range repositoryIDs {
		require.NoError(t, models.EnqueueGitHubAppRepositorySync(db, repositoryID, time.Now().Add(-time.Second)))
	}

	catalog := &recordingGitHubAppCatalog{}
	worker := &GitHubAppCatalogWorker{
		catalog: catalog,
		logger:  log.WithField("worker", "GitHubAppCatalogWorker"),
	}
	worker.processJobs(t.Context())

	assert.ElementsMatch(t, repositoryIDs, catalog.repositoryIDs)
	var remaining int64
	require.NoError(t, db.Model(&models.GitHubAppRepositorySyncJob{}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}
