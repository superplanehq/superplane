package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

type FactoryCleanupWorker struct {
	semaphore           *semaphore.Weighted
	logger              *log.Entry
	maxResourcesPerTick int
}

func NewFactoryCleanupWorker() *FactoryCleanupWorker {
	return &FactoryCleanupWorker{
		semaphore:           semaphore.NewWeighted(10),
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

	if err := deleteFactoryFileObjects(tx, factory.ID, w.maxResourcesPerTick); err != nil {
		return fmt.Errorf("delete factory file objects: %w", err)
	}

	var remainingFiles int64
	if err := tx.Model(&models.File{}).Where("factory_id = ?", factory.ID).Limit(1).Count(&remainingFiles).Error; err != nil {
		return fmt.Errorf("count remaining factory files: %w", err)
	}
	if remainingFiles > 0 {
		w.logger.Infof("Factory %s still has files - waiting for object cleanup", factory.ID)
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

func deleteFactoryFileObjects(tx *gorm.DB, factoryID uuid.UUID, limit int) error {
	files, err := models.ListFilesForFactory(tx, factoryID, limit)
	if err != nil {
		return err
	}
	provider := blob.Current()
	ctx := context.Background()
	for i := range files {
		if err := storedfiles.DeleteObjectAndRow(ctx, tx, provider, &files[i]); err != nil {
			return err
		}
	}
	return nil
}
