package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

type CanvasCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxRunsPerTick      int
	maxResourcesPerTick int
}

func NewCanvasCleanupWorker() *CanvasCleanupWorker {
	return &CanvasCleanupWorker{
		semaphore:           semaphore.NewWeighted(25),
		logger:              log.WithFields(log.Fields{"worker": "CanvasCleanupWorker"}),
		maxRunsPerTick:      50,
		maxResourcesPerTick: 500,
	}
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
			canvases, err := models.ListDeletedCanvases(database.Conn())
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

	totalRunsDeleted, err := w.cleanCanvasRuns(database.Conn(), canvas)
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

	return database.Conn().Transaction(func(tx *gorm.DB) error {
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

		return w.finalizeCanvas(tx, *lockedCanvas)
	})
}

func (w *CanvasCleanupWorker) cleanCanvasRuns(db *gorm.DB, canvas models.Canvas) (int, error) {
	if !canvas.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted canvas %s", canvas.ID)
		return 0, nil
	}

	runs, err := canvas.ListRuns(db, w.maxRunsPerTick)
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
		// used elsewhere (EventRouter, NodeExecutor).
		var summary *models.RunDeletionSummary
		err := db.Transaction(func(tx *gorm.DB) error {
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

func (w *CanvasCleanupWorker) finalizeCanvas(tx *gorm.DB, canvas models.Canvas) error {
	var webhookIDs []uuid.UUID
	if err := tx.Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("webhook_id IS NOT NULL").
		Distinct("webhook_id").
		Pluck("webhook_id", &webhookIDs).Error; err != nil {
		return fmt.Errorf("list canvas webhook ids: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ?", canvas.ID).Delete(&models.CanvasNode{}).Error; err != nil {
		return fmt.Errorf("failed to delete canvas nodes: %w", err)
	}

	for _, webhookID := range webhookIDs {
		if err := models.SoftDeleteWebhookIfUnreferenced(tx, webhookID); err != nil {
			return fmt.Errorf("soft-delete unreferenced webhook %s: %w", webhookID, err)
		}
	}

	// A factory intake holds a RESTRICT reference to its canvas, so the row has
	// to go before the canvas does.
	if err := models.DeleteFactoryIntakesByCanvas(tx, canvas.ID); err != nil {
		return fmt.Errorf("delete factory intakes: %w", err)
	}

	if err := models.DeleteFactoryPRFeedbackHandlersByCanvas(tx, canvas.ID); err != nil {
		return fmt.Errorf("delete factory PR feedback handlers: %w", err)
	}

	if err := tx.Unscoped().Delete(&canvas).Error; err != nil {
		return fmt.Errorf("failed to delete canvas: %w", err)
	}

	logging.WithCanvas(w.logger, canvas).Info("Successfully cleaned up canvas")
	return nil
}
