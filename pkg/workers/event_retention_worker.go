package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage"
)

const (
	eventRetentionBatchSize = 100
	eventRetentionEvery     = 1 * time.Minute
)

type EventRetentionWorker struct {
	semaphore    *semaphore.Weighted
	logger       *log.Entry
	usageService usage.Service
}

func NewEventRetentionWorker(usageService usage.Service) *EventRetentionWorker {
	return &EventRetentionWorker{
		semaphore:    semaphore.NewWeighted(10),
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
	referenceTime := time.Now().UTC()
	rootEvents, err := models.ListExpiredRoutedRootCanvasEvents(referenceTime, eventRetentionBatchSize)
	if err != nil {
		w.logger.Errorf("Error listing expired root events for retention cleanup: %v", err)
		return
	}

	for _, rootEvent := range rootEvents {
		if err := w.semaphore.Acquire(ctx, 1); err != nil {
			if ctx.Err() != nil {
				return
			}

			w.logger.Errorf("Error acquiring semaphore: %v", err)
			continue
		}

		go func(rootEvent models.CanvasEvent, referenceTime time.Time) {
			defer w.semaphore.Release(1)

			if err := w.LockAndProcessRootEvent(rootEvent, referenceTime); err != nil {
				w.logger.Errorf("Error processing retained root event %s: %v", rootEvent.ID, err)
			}
		}(rootEvent, referenceTime)
	}
}

func (w *EventRetentionWorker) LockAndProcessRootEvent(rootEvent models.CanvasEvent, referenceTime time.Time) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedEvent, err := models.LockExpiredRoutedRootCanvasEvent(tx, rootEvent.ID, referenceTime)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.logger.Infof("Root event %s already processed or no longer eligible - skipping", rootEvent.ID)
				return nil
			}

			return fmt.Errorf("lock expired routed root event %s: %w", rootEvent.ID, err)
		}

		return w.processRootEvent(tx, *lockedEvent)
	})
}

func (w *EventRetentionWorker) processRootEvent(tx *gorm.DB, rootEvent models.CanvasEvent) error {
	queueItemsCount, err := models.CountNodeQueueItemsForRootEventInTransaction(tx, rootEvent.ID)
	if err != nil {
		return fmt.Errorf("count queue items for root event %s: %w", rootEvent.ID, err)
	}

	if queueItemsCount > 0 {
		return nil
	}

	activeExecutionsCount, err := models.CountActiveNodeExecutionsForRootEventInTransaction(tx, rootEvent.ID)
	if err != nil {
		return fmt.Errorf("count active executions for root event %s: %w", rootEvent.ID, err)
	}

	if activeExecutionsCount > 0 {
		return nil
	}

	executions, err := models.ListNodeExecutionsForRootEventsInTransaction(tx, []uuid.UUID{rootEvent.ID})
	if err != nil {
		return fmt.Errorf("list executions for root event %s: %w", rootEvent.ID, err)
	}

	executionIDs := make([]uuid.UUID, 0, len(executions))
	for _, execution := range executions {
		executionIDs = append(executionIDs, execution.ID)
	}

	pendingRequestsCount, err := models.CountPendingRequestsForExecutionsInTransaction(tx, executionIDs)
	if err != nil {
		return fmt.Errorf("count pending requests for root event %s: %w", rootEvent.ID, err)
	}

	if pendingRequestsCount > 0 {
		return nil
	}

	if len(executionIDs) > 0 {
		if err := tx.Where("root_event_id = ?", rootEvent.ID).Delete(&models.CanvasNodeExecution{}).Error; err != nil {
			return fmt.Errorf("delete executions for root event %s: %w", rootEvent.ID, err)
		}
	}

	if err := tx.Delete(&rootEvent).Error; err != nil {
		return fmt.Errorf("delete root event %s: %w", rootEvent.ID, err)
	}

	return nil
}
