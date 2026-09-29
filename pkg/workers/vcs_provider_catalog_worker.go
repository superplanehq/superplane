package workers

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	vcsProviderReconcileInterval = 5 * time.Minute
	vcsProviderJobPollInterval   = time.Second
	vcsProviderClaimTimeout      = 5 * time.Minute
	vcsProviderJobBatchSize      = 8
)

type vcsProviderCatalog interface {
	Reconcile(context.Context) error
	SyncRepositoryCollaborators(context.Context, int64) error
}

type VCSProviderCatalogWorker struct {
	provider string
	catalog  vcsProviderCatalog
	logger   *log.Entry
}

func NewVCSProviderCatalogWorker(provider string, catalog vcsProviderCatalog) *VCSProviderCatalogWorker {
	return &VCSProviderCatalogWorker{
		provider: provider,
		catalog:  catalog,
		logger: log.WithFields(log.Fields{
			"worker":   "VCSProviderCatalogWorker",
			"provider": provider,
		}),
	}
}

func (w *VCSProviderCatalogWorker) Start(ctx context.Context) {
	w.reconcile(ctx)

	reconcileTicker := time.NewTicker(vcsProviderReconcileInterval)
	jobTicker := time.NewTicker(vcsProviderJobPollInterval)
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
			w.processJobs(ctx)
		}
	}
}

func (w *VCSProviderCatalogWorker) processReconcileJob(ctx context.Context) {
	now := time.Now()
	job, err := models.ClaimVCSProviderReconciliation(database.Conn(), w.provider, now, now.Add(-vcsProviderClaimTimeout))
	if err != nil {
		w.logger.WithError(err).Error("failed to claim a VCS provider reconciliation job")
		return
	}
	if job == nil {
		return
	}

	err = w.catalog.Reconcile(ctx)
	if err == nil {
		if completeErr := models.CompleteVCSProviderReconciliation(database.Conn(), w.provider); completeErr != nil {
			w.logger.WithError(completeErr).Error("failed to complete a VCS provider reconciliation job")
		}
		return
	}

	if retryErr := models.RetryVCSProviderReconciliation(
		database.Conn(),
		w.provider,
		now.Add(vcsProviderRetryDelay(job.Attempts)),
		err,
	); retryErr != nil {
		w.logger.WithError(retryErr).Error("failed to retry a VCS provider reconciliation job")
	}
}

func (w *VCSProviderCatalogWorker) reconcile(ctx context.Context) {
	if err := w.catalog.Reconcile(ctx); err != nil {
		w.logger.WithError(err).Error("failed to reconcile the VCS provider catalog")
	}
}

func (w *VCSProviderCatalogWorker) processJobs(ctx context.Context) {
	now := time.Now()
	jobs := make([]*models.VCSProviderRepositorySyncJob, 0, vcsProviderJobBatchSize)
	for range vcsProviderJobBatchSize {
		job, err := models.ClaimVCSProviderRepositorySync(database.Conn(), w.provider, now, now.Add(-vcsProviderClaimTimeout))
		if err != nil {
			w.logger.WithError(err).Error("failed to claim a VCS collaborator synchronization job")
			break
		}
		if job == nil {
			break
		}
		jobs = append(jobs, job)
	}

	var group sync.WaitGroup
	for _, job := range jobs {
		group.Add(1)
		go func() {
			defer group.Done()
			w.processClaimedJob(ctx, job, now)
		}()
	}
	group.Wait()
}

func (w *VCSProviderCatalogWorker) processClaimedJob(
	ctx context.Context,
	job *models.VCSProviderRepositorySyncJob,
	claimedAt time.Time,
) {
	err := w.catalog.SyncRepositoryCollaborators(ctx, job.RepositoryID)
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		if completeErr := models.CompleteVCSProviderRepositorySync(database.Conn(), w.provider, job.RepositoryID); completeErr != nil {
			w.logger.WithError(completeErr).Error("failed to complete a VCS collaborator synchronization job")
		}
		return
	}

	if retryErr := models.RetryVCSProviderRepositorySync(
		database.Conn(),
		w.provider,
		job.RepositoryID,
		claimedAt.Add(vcsProviderRetryDelay(job.Attempts)),
		err,
	); retryErr != nil {
		w.logger.WithError(retryErr).Error("failed to retry a VCS collaborator synchronization job")
	}
}

func vcsProviderRetryDelay(attempts int) time.Duration {
	delay := time.Duration(attempts) * time.Minute
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}
