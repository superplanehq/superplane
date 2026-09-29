package workers

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/githubapp"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	githubAppReconcileInterval = 5 * time.Minute
	githubAppJobPollInterval   = time.Second
	githubAppClaimTimeout      = 5 * time.Minute
)

type GitHubAppCatalogWorker struct {
	catalog *githubapp.Catalog
	logger  *log.Entry
}

func NewGitHubAppCatalogWorker(catalog *githubapp.Catalog) *GitHubAppCatalogWorker {
	return &GitHubAppCatalogWorker{
		catalog: catalog,
		logger:  log.WithField("worker", "GitHubAppCatalogWorker"),
	}
}

func (w *GitHubAppCatalogWorker) Start(ctx context.Context) {
	w.reconcile(ctx)

	reconcileTicker := time.NewTicker(githubAppReconcileInterval)
	jobTicker := time.NewTicker(githubAppJobPollInterval)
	defer reconcileTicker.Stop()
	defer jobTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-reconcileTicker.C:
			w.reconcile(ctx)
		case <-jobTicker.C:
			w.processReconcileJob(ctx)
			w.processJob(ctx)
		}
	}
}

func (w *GitHubAppCatalogWorker) processReconcileJob(ctx context.Context) {
	now := time.Now()
	job, err := models.ClaimGitHubAppReconciliation(database.Conn(), now, now.Add(-githubAppClaimTimeout))
	if err != nil {
		w.logger.WithError(err).Error("failed to claim a GitHub App reconciliation job")
		return
	}
	if job == nil {
		return
	}

	err = w.catalog.Reconcile(ctx)
	if err == nil {
		if completeErr := models.CompleteGitHubAppReconciliation(database.Conn()); completeErr != nil {
			w.logger.WithError(completeErr).Error("failed to complete a GitHub App reconciliation job")
		}
		return
	}

	if retryErr := models.RetryGitHubAppReconciliation(
		database.Conn(),
		now.Add(githubAppRetryDelay(job.Attempts)),
		err,
	); retryErr != nil {
		w.logger.WithError(retryErr).Error("failed to retry a GitHub App reconciliation job")
	}
}

func (w *GitHubAppCatalogWorker) reconcile(ctx context.Context) {
	if err := w.catalog.Reconcile(ctx); err != nil {
		w.logger.WithError(err).Error("failed to reconcile the GitHub App catalog")
	}
}

func (w *GitHubAppCatalogWorker) processJob(ctx context.Context) {
	now := time.Now()
	job, err := models.ClaimGitHubAppRepositorySync(database.Conn(), now, now.Add(-githubAppClaimTimeout))
	if err != nil {
		w.logger.WithError(err).Error("failed to claim a GitHub collaborator synchronization job")
		return
	}
	if job == nil {
		return
	}

	err = w.catalog.SyncRepositoryCollaborators(ctx, job.RepositoryID)
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		if completeErr := models.CompleteGitHubAppRepositorySync(database.Conn(), job.RepositoryID); completeErr != nil {
			w.logger.WithError(completeErr).Error("failed to complete a GitHub collaborator synchronization job")
		}
		return
	}

	if retryErr := models.RetryGitHubAppRepositorySync(
		database.Conn(),
		job.RepositoryID,
		now.Add(githubAppRetryDelay(job.Attempts)),
		err,
	); retryErr != nil {
		w.logger.WithError(retryErr).Error("failed to retry a GitHub collaborator synchronization job")
	}
}

func githubAppRetryDelay(attempts int) time.Duration {
	delay := time.Duration(attempts) * time.Minute
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}
