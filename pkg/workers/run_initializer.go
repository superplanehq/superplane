package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type RunInitializer struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	logger    *log.Entry
}

func NewRunInitializer(registry *registry.Registry) *RunInitializer {
	return &RunInitializer{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "RunInitializer"}),
	}
}

func (w *RunInitializer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runs, err := models.ListPendingRuns(database.Conn())
			if err != nil {
				w.logger.Errorf("Error listing pending runs: %v", err)
				continue
			}

			for _, run := range runs {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(run models.CanvasRun) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRun(run); err != nil {
						w.logger.Errorf("Error processing run %s: %v", run.ID, err)
					}
				}(run)
			}
		}
	}
}

func (w *RunInitializer) LockAndProcessRun(run models.CanvasRun) error {
	logger := w.logger.WithFields(log.Fields{"run": run.ID})
	logger.Infof("Locking and processing run")

	newEvents := []models.CanvasEvent{}
	eventCollector := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockCanvasRunInTransaction(tx, run.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Infof("Run already processed - skipping")
				return nil
			}

			return err
		}

		err = NewRunCallbackDispatcher(tx, w.registry, locked).
			WithEventCollector(eventCollector).
			DispatchPending()

		if err != nil {
			return fmt.Errorf("dispatch pending: %w", err)
		}

		if err := locked.Start(tx); err != nil {
			return fmt.Errorf("start run: %w", err)
		}

		newEvents = append(newEvents, newEvents...)
		return nil
	})

	if err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	return nil
}
