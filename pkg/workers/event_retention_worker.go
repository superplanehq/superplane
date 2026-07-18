package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/pkg/workers/cleaners"
)

const (
	eventRetentionBatchSize      = 100
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
	startedAt := time.Now()
	referenceTime := time.Now().UTC()
	deleted, err := w.processRetentionBatches(ctx, referenceTime, eventRetentionMaxRunsPerTick)
	if err != nil {
		w.logger.Errorf("Error processing expired runs for retention cleanup: %v", err)
		return
	}

	if deleted == 0 {
		return
	}

	logger := w.logger.WithFields(log.Fields{
		"deleted_runs": deleted,
		"max_runs":     eventRetentionMaxRunsPerTick,
		"duration_ms":  time.Since(startedAt).Milliseconds(),
	})

	if deleted >= eventRetentionMaxRunsPerTick {
		logger.Warn("Event retention cleanup reached the per-tick limit; more expired runs may remain")
		return
	}

	logger.Info("Deleted retained runs")
}

func (w *EventRetentionWorker) processRetentionBatches(ctx context.Context, referenceTime time.Time, maxRuns int) (int, error) {
	totalDeleted := 0
	for totalDeleted < maxRuns {
		if ctx.Err() != nil {
			return totalDeleted, ctx.Err()
		}

		limit := min(eventRetentionBatchSize, maxRuns-totalDeleted)
		deleted, err := w.cleanRetainedRuns(referenceTime, limit)
		if err != nil {
			return totalDeleted, err
		}

		if deleted == 0 {
			return totalDeleted, nil
		}

		totalDeleted += deleted
	}

	return totalDeleted, nil
}

func (w *EventRetentionWorker) cleanRetainedRuns(referenceTime time.Time, limit int) (int, error) {
	var deleted int
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		runCleaner, err := cleaners.NewRunCleaner(tx, cleaners.RunCleanerOptions{
			Mode:          cleaners.RunCleanerModeRetention,
			ReferenceTime: referenceTime,
		})
		if err != nil {
			return fmt.Errorf("create run cleaner: %w", err)
		}

		deleted, err = runCleaner.CleanBatch(limit)
		if err != nil {
			return err
		}

		return nil
	})

	return deleted, err
}
