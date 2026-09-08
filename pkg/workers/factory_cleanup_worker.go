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

	readyForDomain, err := w.commitFactoryFileCleanup(factory)
	if err != nil || !readyForDomain {
		return err
	}
	return w.commitFactoryDomainCleanup(factory)
}

func (w *FactoryCleanupWorker) commitFactoryFileCleanup(factory models.Factory) (bool, error) {
	canDeleteFiles, err := w.prepareFactoryFileCleanup(factory)
	if err != nil || !canDeleteFiles {
		return false, err
	}
	if err := deleteFactoryFileObjects(factory.ID, w.maxResourcesPerTick); err != nil {
		return false, fmt.Errorf("delete factory file objects: %w", err)
	}
	return w.factoryHasNoFiles(factory)
}

func (w *FactoryCleanupWorker) prepareFactoryFileCleanup(factory models.Factory) (bool, error) {
	var canDeleteFiles bool
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockDeletedFactory(tx, factory.ID)
		if err != nil {
			w.logger.Infof("Factory %s already being processed - skipping", factory.ID)
			return nil
		}

		if err := locked.SoftDeleteCanvases(tx); err != nil {
			return fmt.Errorf("soft delete factory canvases: %w", err)
		}

		remainingCanvases, err := locked.CountCanvases(tx)
		if err != nil {
			return fmt.Errorf("count factory canvases: %w", err)
		}
		if remainingCanvases > 0 {
			w.logger.Infof("Factory %s still has %d canvases - waiting for canvas cleanup", locked.ID, remainingCanvases)
			return nil
		}

		canDeleteFiles = true
		return nil
	})
	return canDeleteFiles, err
}

func (w *FactoryCleanupWorker) factoryHasNoFiles(factory models.Factory) (bool, error) {
	var readyForDomain bool
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockDeletedFactory(tx, factory.ID)
		if err != nil {
			w.logger.Infof("Factory %s already being processed - skipping", factory.ID)
			return nil
		}

		var remainingFiles int64
		if err := tx.Model(&models.File{}).Where("factory_id = ?", locked.ID).Limit(1).Count(&remainingFiles).Error; err != nil {
			return fmt.Errorf("count remaining factory files: %w", err)
		}
		if remainingFiles > 0 {
			w.logger.Infof("Factory %s still has files - waiting for object cleanup", locked.ID)
			return nil
		}

		readyForDomain = true
		return nil
	})
	return readyForDomain, err
}

func (w *FactoryCleanupWorker) commitFactoryDomainCleanup(factory models.Factory) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockDeletedFactory(tx, factory.ID)
		if err != nil {
			w.logger.Infof("Factory %s already being processed - skipping", factory.ID)
			return nil
		}

		deleted, complete, err := models.NewFactoryResourceCleaner(tx, locked).
			WithLimit(w.maxResourcesPerTick).
			Run()
		if err != nil {
			return err
		}

		if !complete {
			w.logger.Infof("Partially cleaned factory %s (deleted %d rows this tick)", locked.ID, deleted)
			return nil
		}

		w.logger.Infof("Successfully cleaned up factory %s", locked.ID)
		return nil
	})
}

func deleteFactoryFileObjects(factoryID uuid.UUID, limit int) error {
	files, err := models.ListFilesForFactory(database.Conn(), factoryID, limit)
	if err != nil {
		return err
	}
	provider := blob.Current()
	ctx := context.Background()
	for i := range files {
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			return storedfiles.DeleteObjectAndRow(ctx, tx, provider, &files[i])
		})
		if err != nil {
			return err
		}
	}
	return nil
}
