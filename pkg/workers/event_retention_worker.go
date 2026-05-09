package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage"
)

const (
	//
	// eventRetentionBatchSize bounds how many root events are locked, walked
	// and deleted in a single transaction. Each root event can fan out to
	// many related rows (executions, requests, kvs, child events), so a large
	// batch keeps the transaction open for longer and increases the chance of
	// hitting `statement_timeout` (60s by default) on one of the cascading
	// deletes — which surfaces in Sentry as
	// `pgconn.PgError: ERROR: canceling statement due to statement timeout (SQLSTATE 57014)`.
	//
	// Keep this small enough that one batch comfortably fits inside the
	// statement timeout even on busy canvases. Throughput is preserved by
	// looping more times per tick (`eventRetentionMaxRootEventsPerTick`).
	//
	eventRetentionBatchSize            = 25
	eventRetentionMaxRootEventsPerTick = 1000
	eventRetentionEvery                = 1 * time.Minute
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
	deleted, err := w.processRetentionBatches(ctx, referenceTime, eventRetentionMaxRootEventsPerTick)
	if err != nil {
		w.logger.Errorf("Error processing expired root events for retention cleanup: %v", err)
		return
	}

	if deleted == 0 {
		return
	}

	logger := w.logger.WithFields(log.Fields{
		"deleted_root_events": deleted,
		"max_root_events":     eventRetentionMaxRootEventsPerTick,
		"duration_ms":         time.Since(startedAt).Milliseconds(),
	})

	if deleted >= eventRetentionMaxRootEventsPerTick {
		logger.Warn("Event retention cleanup reached the per-tick limit; more expired root events may remain")
		return
	}

	logger.Info("Deleted retained root events")
}

func (w *EventRetentionWorker) processRetentionBatches(ctx context.Context, referenceTime time.Time, maxRootEvents int) (int, error) {
	totalDeleted := 0
	for totalDeleted < maxRootEvents {
		if ctx.Err() != nil {
			return totalDeleted, ctx.Err()
		}

		limit := min(eventRetentionBatchSize, maxRootEvents-totalDeleted)
		deleted, err := w.LockAndProcessRootEvents(referenceTime, limit)
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

func (w *EventRetentionWorker) LockAndProcessRootEvents(referenceTime time.Time, limit int) (int, error) {
	var deleted int
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		events, err := models.LockExpiredRoutedRootCanvasEventsInTransaction(tx, referenceTime, limit)
		if err != nil {
			return fmt.Errorf("lock expired routed root events: %w", err)
		}

		if len(events) == 0 {
			return nil
		}

		rootEventIDs := make([]uuid.UUID, 0, len(events))
		for _, event := range events {
			rootEventIDs = append(rootEventIDs, event.ID)
		}

		if err := models.DeleteRootCanvasEventChainsInTransaction(tx, rootEventIDs); err != nil {
			return fmt.Errorf("delete root event chains: %w", err)
		}

		deleted = len(rootEventIDs)
		return nil
	})

	return deleted, err
}
