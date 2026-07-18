package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type CanvasCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxRunsPerTick      int
	maxResourcesPerTick int
	sessionCleaner      agents.ProviderSessionCleaner
	gitProvider         git.Provider
}

func NewCanvasCleanupWorker(gitProvider git.Provider, providers ...agents.Provider) *CanvasCleanupWorker {
	w := &CanvasCleanupWorker{
		semaphore:           semaphore.NewWeighted(25),
		logger:              log.WithFields(log.Fields{"worker": "CanvasCleanupWorker"}),
		maxRunsPerTick:      50,
		maxResourcesPerTick: 500,
		gitProvider:         gitProvider,
	}

	if len(providers) > 0 {
		if cleaner, ok := providers[0].(agents.ProviderSessionCleaner); ok {
			w.sessionCleaner = cleaner
		}
	}

	return w
}

func (w *CanvasCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			canvases, err := models.ListDeletedCanvases()
			if err != nil {
				w.logger.Errorf("Error finding deleted canvases: %v", err)
				continue
			}

			w.logger.Infof("Found %d deleted canvases for cleanup", len(canvases))
			telemetry.RecordWorkflowCleanupWorkerCanvasesCount(context.Background(), len(canvases))

			for _, canvas := range canvases {
				if deletedResourceWithinGracePeriod(canvas.DeletedAt.Time, tickStart) {
					continue
				}

				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(canvas models.Canvas) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessCanvas(canvas); err != nil {
						w.logger.Errorf("Error processing canvas %s: %v", canvas.ID, err)
					}
				}(canvas)
			}

			telemetry.RecordWorkflowCleanupWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *CanvasCleanupWorker) LockAndProcessCanvas(canvas models.Canvas) error {
	if deletedResourceWithinGracePeriod(canvas.DeletedAt.Time, time.Now()) {
		return nil
	}

	w.logger.Infof("Processing deleted canvas %s", canvas.ID)

	totalRunsDeleted, err := w.cleanCanvasRuns(canvas)
	if err != nil {
		return err
	}

	remainingRuns, err := canvas.CountRuns(database.Conn())
	if err != nil {
		return fmt.Errorf("count remaining workflow runs: %w", err)
	}

	if remainingRuns > 0 {
		w.logger.Infof("Partially cleaned runs from canvas %s (deleted %d runs, %d remaining)", canvas.ID, totalRunsDeleted, remainingRuns)
		return nil
	}

	// Run-scoped rows are removed first. DeleteRemainingResources then sweeps
	// workflow rows that are not tied to a run (see Canvas.DeleteRemainingResources).
	complete, err := w.cleanRemainingResources(canvas)
	if err != nil {
		return err
	}

	if !complete {
		w.logger.Infof("Reached max resources per tick (%d), stopping remaining resource cleanup for canvas %s", w.maxResourcesPerTick, canvas.ID)
		return nil
	}

	var sessionsToClean []models.AgentSession
	var repositoriesToClean []models.Repository
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedCanvas, err := models.LockCanvas(tx, canvas.ID)
		if err != nil {
			w.logger.Infof("Canvas %s already being finalized - skipping", canvas.ID)
			return nil
		}

		remainingRuns, err := lockedCanvas.CountRuns(tx)
		if err != nil {
			return fmt.Errorf("count remaining workflow runs: %w", err)
		}

		if remainingRuns > 0 {
			return nil
		}

		sessions, repositories, err := w.finalizeCanvas(tx, *lockedCanvas)
		if err != nil {
			return err
		}

		sessionsToClean = sessions
		repositoriesToClean = repositories
		return nil
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	w.cleanupProviderSessions(ctx, sessionsToClean)
	w.cleanupGitRepositories(ctx, repositoriesToClean)
	return nil
}

func (w *CanvasCleanupWorker) cleanCanvasRuns(canvas models.Canvas) (int, error) {
	if !canvas.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted canvas %s", canvas.ID)
		return 0, nil
	}

	runs, err := canvas.ListRuns(database.Conn(), w.maxRunsPerTick)
	if err != nil {
		return 0, fmt.Errorf("list workflow runs for cleanup: %w", err)
	}

	if len(runs) == 0 {
		return 0, nil
	}

	deleted := 0
	for _, run := range runs {
		logger := logging.WithRun(w.logger, run)

		// One transaction per run, not one transaction for the whole batch. A run
		// delete must be atomic, but batching many runs in a single transaction
		// risks statement timeouts and rolls back all progress on failure; per-run
		// transactions commit incrementally and match the list-then-lock pattern
		// used elsewhere (EventRouter, NodeExecutor, EventRetentionWorker).
		var summary *models.RunDeletionSummary
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			locked, err := models.LockCanvasRun(tx, canvas.ID, run.ID)
			if err != nil {
				return fmt.Errorf("lock run %s: %w", run.ID, err)
			}

			if locked == nil {
				return nil
			}

			summary, err = locked.DeleteChain(tx)
			if err != nil {
				return fmt.Errorf("delete run chain: %w", err)
			}

			return nil
		})

		if err != nil {
			logger.Errorf("Error cleaning run: %v", err)
			return deleted, err
		}

		if summary != nil {
			logger.WithFields(log.Fields{
				"runs":          summary.Runs,
				"events":        summary.Events,
				"executions":    summary.NodeExecutions,
				"requests":      summary.NodeRequests,
				"execution_kvs": summary.NodeExecutionKVs,
				"queue_items":   summary.NodeQueueItems,
			}).Info("Cleaned run")
			deleted++
		}
	}

	return deleted, nil
}

func (w *CanvasCleanupWorker) cleanRemainingResources(canvas models.Canvas) (bool, error) {
	var complete bool

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		summary, done, err := canvas.DeleteRemainingResources(tx, w.maxResourcesPerTick)
		if err != nil {
			return err
		}

		complete = done

		if summary != nil && summary.TotalRecords() > 0 {
			w.logger.WithFields(log.Fields{
				"canvas_id":     canvas.ID,
				"runs":          summary.Runs,
				"events":        summary.Events,
				"executions":    summary.NodeExecutions,
				"requests":      summary.NodeRequests,
				"execution_kvs": summary.NodeExecutionKVs,
				"queue_items":   summary.NodeQueueItems,
			}).Info("Cleaned remaining workflow resources")
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("delete remaining workflow resources: %w", err)
	}

	return complete, nil
}

func (w *CanvasCleanupWorker) finalizeCanvas(tx *gorm.DB, canvas models.Canvas) ([]models.AgentSession, []models.Repository, error) {
	if err := tx.Unscoped().Where("workflow_id = ?", canvas.ID).Delete(&models.CanvasNode{}).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to delete canvas nodes: %w", err)
	}

	sessions, err := models.ListAgentSessionsForCanvasInTransaction(tx, canvas.OrganizationID, canvas.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list canvas agent sessions: %w", err)
	}

	var repositories []models.Repository
	repository, err := models.FindRepositoryInTransaction(tx, canvas.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, fmt.Errorf("find canvas repository: %w", err)
	}

	if repository != nil {
		repositories = append(repositories, *repository)
	}

	if err := models.DeleteAgentSessionsForCanvasInTransaction(tx, canvas.OrganizationID, canvas.ID); err != nil {
		return nil, nil, fmt.Errorf("delete canvas agent sessions: %w", err)
	}

	if err := tx.Unscoped().Delete(&canvas).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to delete canvas: %w", err)
	}

	w.logger.WithFields(log.Fields{
		"canvas_id": canvas.ID,
	}).Info("Successfully cleaned up canvas")
	return sessions, repositories, nil
}

func (w *CanvasCleanupWorker) processCanvas(tx *gorm.DB, canvas models.Canvas) ([]models.AgentSession, []models.Repository, error) {
	if !canvas.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted canvas %s", canvas.ID)
		return nil, nil, nil
	}

	if _, err := w.cleanCanvasRuns(canvas); err != nil {
		return nil, nil, err
	}

	remainingRuns, err := canvas.CountRuns(tx)
	if err != nil {
		return nil, nil, fmt.Errorf("count remaining workflow runs: %w", err)
	}

	if remainingRuns > 0 {
		return nil, nil, nil
	}

	summary, complete, err := canvas.DeleteRemainingResources(tx, w.maxResourcesPerTick)
	if err != nil {
		return nil, nil, fmt.Errorf("delete remaining workflow resources: %w", err)
	}

	if !complete {
		if summary != nil && summary.TotalRecords() > 0 {
			w.logger.WithFields(log.Fields{
				"canvas_id":     canvas.ID,
				"runs":          summary.Runs,
				"events":        summary.Events,
				"executions":    summary.NodeExecutions,
				"requests":      summary.NodeRequests,
				"execution_kvs": summary.NodeExecutionKVs,
				"queue_items":   summary.NodeQueueItems,
			}).Info("Partially cleaned remaining workflow resources")
		}

		return nil, nil, nil
	}

	if summary != nil && summary.TotalRecords() > 0 {
		w.logger.WithFields(log.Fields{
			"canvas_id":     canvas.ID,
			"runs":          summary.Runs,
			"events":        summary.Events,
			"executions":    summary.NodeExecutions,
			"requests":      summary.NodeRequests,
			"execution_kvs": summary.NodeExecutionKVs,
			"queue_items":   summary.NodeQueueItems,
		}).Info("Cleaned remaining workflow resources")
	}

	lockedCanvas, err := models.LockCanvas(tx, canvas.ID)
	if err != nil {
		return nil, nil, nil
	}

	remainingRuns, err = lockedCanvas.CountRuns(tx)
	if err != nil {
		return nil, nil, fmt.Errorf("count remaining workflow runs: %w", err)
	}

	if remainingRuns > 0 {
		return nil, nil, nil
	}

	return w.finalizeCanvas(tx, *lockedCanvas)
}

func (w *CanvasCleanupWorker) cleanupProviderSessions(ctx context.Context, sessions []models.AgentSession) {
	if w.sessionCleaner == nil || len(sessions) == 0 {
		return
	}

	for _, session := range sessions {
		if session.Provider != w.sessionCleaner.Name() {
			w.logger.WithFields(log.Fields{
				"session_id":       session.ID,
				"session_provider": session.Provider,
				"cleaner_provider": w.sessionCleaner.Name(),
			}).Warn("Skipping provider cleanup for agent session with mismatched provider")
			continue
		}

		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := w.sessionCleaner.DeleteSession(cleanupCtx, session.ProviderSessionID)
		cancel()
		if err != nil {
			w.logger.WithFields(log.Fields{
				"session_id":          session.ID,
				"provider":            session.Provider,
				"provider_session_id": session.ProviderSessionID,
			}).WithError(err).Warn("Failed to cleanup provider agent session")
		}
	}
}

func (w *CanvasCleanupWorker) cleanupGitRepositories(ctx context.Context, repositories []models.Repository) {
	if w.gitProvider == nil || len(repositories) == 0 {
		return
	}

	for _, repository := range repositories {
		if repository.Provider != w.gitProvider.Name() {
			w.logger.WithFields(log.Fields{
				"repository_id":       repository.ID,
				"repository_provider": repository.Provider,
				"git_provider":        w.gitProvider.Name(),
			}).Warn("Skipping repository cleanup for repository with mismatched provider")
			continue
		}

		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := w.gitProvider.DeleteRepository(cleanupCtx, repository.RepoID)
		cancel()
		if err != nil {
			w.logger.WithFields(log.Fields{
				"repository_id": repository.ID,
				"provider":      repository.Provider,
				"repo_id":       repository.RepoID,
			}).WithError(err).Warn("Failed to cleanup git repository")
		}
	}
}
