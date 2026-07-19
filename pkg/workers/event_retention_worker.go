package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage"
)

const (
	eventRetentionMaxRunsPerTick = 1000
	eventRetentionEvery          = 1 * time.Minute
)

type EventRetentionWorker struct {
	logger       *log.Entry
	usageService usage.Service
}

func NewEventRetentionWorker(usageService usage.Service) *EventRetentionWorker {
	return &EventRetentionWorker{
		logger:       log.WithFields(log.Fields{"worker": "EventRetentionWorker"}),
		usageService: usageService,
	}
}

func (w *EventRetentionWorker) Start(ctx context.Context) {
	if w.usageService == nil || !w.usageService.Enabled() {
		w.logger.Info("Event retention worker not started because usage is disabled")
		return
	}

	w.tick(ctx)

	ticker := time.NewTicker(eventRetentionEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *EventRetentionWorker) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	startedAt := time.Now()
	referenceTime := time.Now().UTC()
	deleted, err := w.cleanRuns(referenceTime, eventRetentionMaxRunsPerTick)
	if err != nil {
		w.logger.Errorf("Error processing expired runs for retention cleanup: %v", err)
		return
	}

	if deleted == 0 {
		w.logger.Info("No runs found for retention cleanup")
		return
	}

	logger := w.logger.WithFields(log.Fields{
		"deleted":  deleted,
		"duration": time.Since(startedAt).String(),
	})

	if deleted >= eventRetentionMaxRunsPerTick {
		logger.Warn("Event retention cleanup reached the per-tick limit; more expired runs may remain")
		return
	}

	logger.Info("Deleted runs")
}

func (w *EventRetentionWorker) cleanRuns(referenceTime time.Time, limit int) (int, error) {
	runs, err := models.ListExpiredFinishedRuns(database.Conn(), referenceTime, limit)
	if err != nil {
		return 0, err
	}

	if len(runs) == 0 {
		return 0, nil
	}

	w.logger.Infof("Found %d runs for cleanup outside the retention window", len(runs))

	deleted := 0
	for _, run := range runs {
		logger := logging.WithRun(w.logger, run)

		var summary *models.RunDeletionSummary
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			locked, err := models.LockExpiredFinishedRun(tx, referenceTime, run.ID)
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
			logger.Errorf("Error deleting run: %v", err)
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
			}).Info("Deleted run")
			deleted++
		}
	}

	return deleted, nil
}
