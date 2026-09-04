package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type FactoryCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxResourcesPerTick int
}

func NewFactoryCleanupWorker() *FactoryCleanupWorker {
	return &FactoryCleanupWorker{
		semaphore:           semaphore.NewWeighted(maxConcurrentCleanupTasks),
		logger:              log.WithFields(log.Fields{"worker": "FactoryCleanupWorker"}),
		maxResourcesPerTick: 500,
	}
}

func (w *FactoryCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			drainTasks(w.semaphore, maxConcurrentCleanupTasks)
			return
		case tickTime := <-ticker.C:
			factories, err := models.ListDeletedFactories(database.Conn())
			if err != nil {
				w.logger.Errorf("Error finding deleted factories: %v", err)
				continue
			}

			w.logger.Infof("Found %d deleted factories for cleanup", len(factories))

			for _, factory := range factories {
				if !factory.DeletedAt.Valid || deletedResourceWithinGracePeriod(factory.DeletedAt.Time, tickTime) {
					continue
				}

				//
				// Stop handing out new work once shutdown starts. Without this the
				// worker keeps launching the rest of the batch after cancellation,
				// and the drain then waits for work it should never have started.
				//
				if ctx.Err() != nil {
					break
				}

				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(factory models.Factory) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessFactory(factory); err != nil {
						w.logger.Errorf("Error processing factory %s: %v", factory.ID, err)
					}
				}(factory)
			}
		}
	}
}

func (w *FactoryCleanupWorker) LockAndProcessFactory(factory models.Factory) error {
	if !factory.DeletedAt.Valid || deletedResourceWithinGracePeriod(factory.DeletedAt.Time, time.Now()) {
		return nil
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockDeletedFactory(tx, factory.ID)
		if err != nil {
			w.logger.Infof("Factory %s already being processed - skipping", factory.ID)
			return nil
		}

		return w.processFactory(tx, locked)
	})
}

func (w *FactoryCleanupWorker) processFactory(tx *gorm.DB, factory *models.Factory) error {
	if err := factory.SoftDeleteCanvases(tx); err != nil {
		return fmt.Errorf("soft delete factory canvases: %w", err)
	}

	remainingCanvases, err := factory.CountCanvases(tx)
	if err != nil {
		return fmt.Errorf("count factory canvases: %w", err)
	}
	if remainingCanvases > 0 {
		w.logger.Infof("Factory %s still has %d canvases - waiting for canvas cleanup", factory.ID, remainingCanvases)
		return nil
	}

	deleted, complete, err := models.NewFactoryResourceCleaner(tx, factory).
		WithLimit(w.maxResourcesPerTick).
		Run()
	if err != nil {
		return err
	}

	if !complete {
		w.logger.Infof("Partially cleaned factory %s (deleted %d rows this tick)", factory.ID, deleted)
		return nil
	}

	w.logger.Infof("Successfully cleaned up factory %s", factory.ID)
	return nil
}
