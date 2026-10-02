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
	vcsProviderReconcileInterval             = 5 * time.Minute
	vcsProviderInstallRequestPollInterval    = 5 * time.Second
	vcsProviderInstallRequestRefreshInterval = 15 * time.Second
	vcsProviderInstallationRequesterGrace    = 5 * time.Minute
	vcsProviderJobPollInterval               = time.Second
	vcsProviderClaimTimeout                  = 5 * time.Minute
	vcsProviderJobBatchSize                  = 8
	vcsProviderInstallationJobConcurrency    = 4
	vcsProviderInstallationMaxAttempts       = 10
	vcsProviderInstallationRetryMaximumDelay = 30 * time.Second
)

type vcsProviderCatalog interface {
	Reconcile(context.Context, models.VCSProviderRepositorySyncPriority) error
	ReconcileInstallRequests(context.Context) error
	ReconcileInstallation(context.Context, int64, models.VCSProviderRepositorySyncPriority) error
	SyncRepositoryCollaborators(context.Context, int64) error
}

type VCSProviderCatalogWorker struct {
	provider                  string
	catalog                   vcsProviderCatalog
	logger                    *log.Entry
	installationSlots         chan struct{}
	installationJobs          sync.WaitGroup
	repositorySlots           chan struct{}
	repositoryJobs            sync.WaitGroup
	lastInstallRequestRefresh time.Time
}

func NewVCSProviderCatalogWorker(provider string, catalog vcsProviderCatalog) *VCSProviderCatalogWorker {
	return &VCSProviderCatalogWorker{
		provider: provider,
		catalog:  catalog,
		logger: log.WithFields(log.Fields{
			"worker":   "VCSProviderCatalogWorker",
			"provider": provider,
		}),
		installationSlots: make(chan struct{}, vcsProviderInstallationJobConcurrency),
		repositorySlots:   make(chan struct{}, vcsProviderJobBatchSize),
	}
}

func (w *VCSProviderCatalogWorker) Start(ctx context.Context) {
	var loops sync.WaitGroup
	loops.Add(3)
	go func() {
		defer loops.Done()
		w.startReconciliation(ctx)
	}()
	go func() {
		defer loops.Done()
		w.startInstallationReconciliation(ctx)
	}()
	go func() {
		defer loops.Done()
		w.startInstallRequestRefresh(ctx)
	}()

	w.startRepositorySynchronization(ctx)
	loops.Wait()
}

// GitHub sends no webhook when a member requests or cancels an approval, and
// its request list can lag behind the setup redirect. The worker reads the
// list often while a refresh is requested, and less often while any request
// is pending.
func (w *VCSProviderCatalogWorker) startInstallRequestRefresh(ctx context.Context) {
	ticker := time.NewTicker(vcsProviderInstallRequestPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			w.refreshInstallRequests(ctx, now)
		}
	}
}

func (w *VCSProviderCatalogWorker) refreshInstallRequests(ctx context.Context, now time.Time) {
	if !w.installRequestRefreshDue(now) {
		return
	}
	if err := w.catalog.ReconcileInstallRequests(ctx); err != nil {
		w.logger.WithError(err).Error("failed to refresh VCS installation requests")
		return
	}
	w.lastInstallRequestRefresh = now
}

func (w *VCSProviderCatalogWorker) installRequestRefreshDue(now time.Time) bool {
	db := database.Conn()
	requested, err := models.VCSProviderInstallRequestRefreshRequested(db, w.provider, now)
	if err != nil {
		w.logger.WithError(err).Error("failed to check requested VCS installation request refresh")
		return false
	}
	if requested {
		return true
	}
	if now.Sub(w.lastInstallRequestRefresh) < vcsProviderInstallRequestRefreshInterval {
		return false
	}
	pending, err := models.HasVCSProviderInstallRequests(db, w.provider)
	if err != nil {
		w.logger.WithError(err).Error("failed to check pending VCS installation requests")
		return false
	}
	return pending
}

func (w *VCSProviderCatalogWorker) startInstallationReconciliation(ctx context.Context) {
	w.processInstallationJobs(ctx)

	jobTicker := time.NewTicker(vcsProviderJobPollInterval)
	defer jobTicker.Stop()
	defer w.installationJobs.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case <-jobTicker.C:
			w.processInstallationJobs(ctx)
		}
	}
}

func (w *VCSProviderCatalogWorker) startReconciliation(ctx context.Context) {
	w.reconcile(ctx)
	w.processReconcileJob(ctx)
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
		}
	}
}

func (w *VCSProviderCatalogWorker) startRepositorySynchronization(ctx context.Context) {
	w.processJobs(ctx)

	jobTicker := time.NewTicker(vcsProviderJobPollInterval)
	defer jobTicker.Stop()
	defer w.repositoryJobs.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case <-jobTicker.C:
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

	err = w.catalog.Reconcile(ctx, models.VCSProviderRepositorySyncPriorityInteractive)
	if err == nil {
		if completeErr := models.CompleteVCSProviderReconciliation(database.Conn(), w.provider, *job.LockedAt); completeErr != nil {
			w.logger.WithError(completeErr).Error("failed to complete a VCS provider reconciliation job")
		}
		return
	}

	if retryErr := models.RetryVCSProviderReconciliation(
		database.Conn(),
		w.provider,
		*job.LockedAt,
		now.Add(vcsProviderRetryDelay(job.Attempts)),
		err,
	); retryErr != nil {
		w.logger.WithError(retryErr).Error("failed to retry a VCS provider reconciliation job")
	}
}

func (w *VCSProviderCatalogWorker) reconcile(ctx context.Context) {
	if err := w.catalog.Reconcile(ctx, models.VCSProviderRepositorySyncPriorityBackground); err != nil {
		w.logger.WithError(err).Error("failed to reconcile the VCS provider catalog")
	}
	w.deleteFinishedInstallationRequesters()
}

func (w *VCSProviderCatalogWorker) deleteFinishedInstallationRequesters() {
	createdBefore := time.Now().Add(-vcsProviderInstallationRequesterGrace)
	if err := models.DeleteFinishedVCSProviderInstallationRequesters(database.Conn(), w.provider, createdBefore); err != nil {
		w.logger.WithError(err).Error("failed to delete finished VCS installation requesters")
	}
}

func (w *VCSProviderCatalogWorker) processInstallationJobs(ctx context.Context) {
	availableSlots := cap(w.installationSlots) - len(w.installationSlots)
	for range availableSlots {
		select {
		case w.installationSlots <- struct{}{}:
		default:
			return
		}

		now := time.Now()
		job, err := models.ClaimVCSProviderInstallationReconciliation(
			database.Conn(),
			w.provider,
			now,
			now.Add(-vcsProviderClaimTimeout),
		)
		if err != nil {
			<-w.installationSlots
			w.logger.WithError(err).Error("failed to claim a VCS installation reconciliation job")
			return
		}
		if job == nil {
			<-w.installationSlots
			return
		}

		w.installationJobs.Add(1)
		go func(job *models.VCSProviderInstallationReconcileJob) {
			defer w.installationJobs.Done()
			defer func() { <-w.installationSlots }()
			w.processClaimedInstallationJob(ctx, job)
		}(job)
	}
}

func (w *VCSProviderCatalogWorker) processClaimedInstallationJob(
	ctx context.Context,
	job *models.VCSProviderInstallationReconcileJob,
) {
	startedAt := time.Now()
	fields := log.Fields{
		"installation_id": job.InstallationID,
		"attempt":         job.Attempts,
		"queue_delay_ms":  max(startedAt.Sub(job.RunAt).Milliseconds(), int64(0)),
	}
	err := w.catalog.ReconcileInstallation(
		ctx,
		job.InstallationID,
		models.VCSProviderRepositorySyncPriorityInteractive,
	)
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		if completeErr := models.CompleteVCSProviderInstallationReconciliation(
			database.Conn(),
			w.provider,
			job.InstallationID,
			*job.LockedAt,
		); completeErr != nil {
			w.logger.WithFields(fields).WithError(completeErr).Error("failed to complete a VCS installation reconciliation job")
			return
		}
		fields["duration_ms"] = time.Since(startedAt).Milliseconds()
		w.logger.WithFields(fields).Info("reconciled VCS installation")
		return
	}
	if job.Attempts >= vcsProviderInstallationMaxAttempts {
		fields["duration_ms"] = time.Since(startedAt).Milliseconds()
		if completeErr := models.CompleteVCSProviderInstallationReconciliation(
			database.Conn(),
			w.provider,
			job.InstallationID,
			*job.LockedAt,
		); completeErr != nil {
			w.logger.WithFields(fields).WithError(completeErr).Error("failed to discard a VCS installation reconciliation job")
			return
		}
		w.logger.WithFields(fields).WithError(err).Error("discarded a VCS installation reconciliation job after repeated failures")
		return
	}

	retryAt := time.Now().Add(vcsProviderInstallationRetryDelay(job.Attempts))
	fields["duration_ms"] = time.Since(startedAt).Milliseconds()
	fields["retry_at"] = retryAt
	if retryErr := models.RetryVCSProviderInstallationReconciliation(
		database.Conn(),
		w.provider,
		job.InstallationID,
		*job.LockedAt,
		retryAt,
		err,
	); retryErr != nil {
		w.logger.WithFields(fields).WithError(retryErr).Error("failed to retry a VCS installation reconciliation job")
		return
	}
	w.logger.WithFields(fields).WithError(err).Warn("VCS installation reconciliation will retry")
}

func (w *VCSProviderCatalogWorker) processJobs(ctx context.Context) {
	availableSlots := cap(w.repositorySlots) - len(w.repositorySlots)
	for range availableSlots {
		select {
		case w.repositorySlots <- struct{}{}:
		default:
			return
		}

		now := time.Now()
		job, err := models.ClaimVCSProviderRepositorySync(database.Conn(), w.provider, now, now.Add(-vcsProviderClaimTimeout))
		if err != nil {
			<-w.repositorySlots
			w.logger.WithError(err).Error("failed to claim a VCS collaborator synchronization job")
			return
		}
		if job == nil {
			<-w.repositorySlots
			return
		}

		w.repositoryJobs.Add(1)
		go func(job *models.VCSProviderRepositorySyncJob, claimedAt time.Time) {
			defer w.repositoryJobs.Done()
			defer func() { <-w.repositorySlots }()
			w.processClaimedJob(ctx, job, claimedAt)
		}(job, now)
	}
}

func (w *VCSProviderCatalogWorker) processClaimedJob(
	ctx context.Context,
	job *models.VCSProviderRepositorySyncJob,
	claimedAt time.Time,
) {
	startedAt := time.Now()
	fields := log.Fields{
		"repository_id": job.RepositoryID,
		"priority":      job.Priority,
		"attempt":       job.Attempts,
		"queue_delay_ms": max(
			startedAt.Sub(job.RunAt).Milliseconds(),
			int64(0),
		),
	}
	err := w.catalog.SyncRepositoryCollaborators(ctx, job.RepositoryID)
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		if completeErr := models.CompleteVCSProviderRepositorySync(
			database.Conn(),
			w.provider,
			job.RepositoryID,
			*job.LockedAt,
		); completeErr != nil {
			w.logger.WithFields(fields).WithError(completeErr).Error("failed to complete a VCS collaborator synchronization job")
			return
		}
		fields["duration_ms"] = time.Since(startedAt).Milliseconds()
		w.logger.WithFields(fields).Info("synchronized VCS repository collaborators")
		return
	}

	retryAt := time.Now().Add(vcsProviderRetryDelay(job.Attempts))
	fields["duration_ms"] = time.Since(startedAt).Milliseconds()
	fields["retry_at"] = retryAt
	if retryErr := models.RetryVCSProviderRepositorySync(
		database.Conn(),
		w.provider,
		job.RepositoryID,
		*job.LockedAt,
		retryAt,
		err,
	); retryErr != nil {
		w.logger.WithFields(fields).WithError(retryErr).Error("failed to retry a VCS collaborator synchronization job")
		return
	}
	w.logger.WithFields(fields).WithError(err).Warn("VCS repository collaborator synchronization will retry")
}

func vcsProviderRetryDelay(attempts int) time.Duration {
	delay := time.Duration(attempts) * time.Minute
	if delay > 15*time.Minute {
		return 15 * time.Minute
	}
	return delay
}

func vcsProviderInstallationRetryDelay(attempts int) time.Duration {
	shift := min(max(attempts-1, 0), 5)
	delay := time.Second << shift
	if delay > vcsProviderInstallationRetryMaximumDelay {
		return vcsProviderInstallationRetryMaximumDelay
	}
	return delay
}
