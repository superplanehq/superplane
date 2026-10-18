package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type ApplicationEncryptionMigrationWorker struct {
	encryptor *crypto.VersionedEncryptor
	registry  *registry.Registry
	locations []models.EncryptedStorageLocation
}

func NewApplicationEncryptionMigrationWorker(encryptor *crypto.VersionedEncryptor, registry *registry.Registry, locations []models.EncryptedStorageLocation) *ApplicationEncryptionMigrationWorker {
	return &ApplicationEncryptionMigrationWorker{encryptor: encryptor, registry: registry, locations: locations}
}

// Start lets every worker pod attempt one short batch at a time. PostgreSQL
// releases the job row lock after each batch or when a pod exits.
func (w *ApplicationEncryptionMigrationWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := w.RunBatch(ctx); err != nil && ctx.Err() == nil {
			log.WithError(err).Error("application encryption migration batch failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *ApplicationEncryptionMigrationWorker) RunBatch(ctx context.Context) error {
	var job *models.ApplicationEncryptionMigration
	var stateBefore, locationBefore string
	var issues []error
	var processingError error
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		job, err = models.LockNextApplicationEncryptionMigration(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			job = nil
			return nil
		}
		if err != nil {
			return err
		}
		stateBefore, locationBefore = job.State, job.StorageLocation
		processingError = tx.Transaction(func(batch *gorm.DB) error {
			if job.State == "auditing" {
				issues, err = encryption.NewAuditor(job, w.registry, w.locations).ProcessBatch(batch)
			} else {
				issues, err = encryption.NewMigrator(job, w.encryptor, w.registry, w.locations).ProcessBatch(ctx, batch)
			}
			return err
		})
		if processingError != nil {
			job.State = "failed"
			job.ErrorMessage = processingError.Error()
			return tx.Model(&models.ApplicationEncryptionMigration{}).
				Where("id = ?", job.ID).
				Updates(map[string]any{"state": job.State, "error_message": job.ErrorMessage}).Error
		}
		return tx.Save(job).Error
	})
	if err != nil {
		if job != nil {
			return fmt.Errorf("migration %s: %w", job.ID, err)
		}
		return err
	}
	if job != nil {
		logEncryptionMigrationBatch(job, stateBefore, locationBefore, issues, processingError)
	}
	return nil
}

func logEncryptionMigrationBatch(job *models.ApplicationEncryptionMigration, stateBefore, locationBefore string, issues []error, processingError error) {
	phase := "migration"
	if stateBefore == "auditing" {
		phase = "audit"
	}
	entry := log.WithFields(log.Fields{
		"migration_id":     job.ID,
		"target_key_id":    job.TargetKeyID,
		"storage_location": locationBefore,
	})
	if processingError != nil {
		entry.WithError(processingError).Error("application encryption " + phase + " failed")
		return
	}
	for _, issue := range issues {
		if stateBefore == "auditing" {
			entry.WithError(issue).Error("application encryption audit found a non-target value")
		} else {
			entry.WithError(issue).Error("application encryption value could not be migrated")
		}
	}
	if job.Cursor == "" {
		completed := entry
		if job.StorageLocation != locationBefore {
			completed = completed.WithField("next_storage_location", job.StorageLocation)
		}
		completed.Info("application encryption " + phase + " location completed")
	}
	if job.State == "failed" {
		entry.WithField("error_message", job.ErrorMessage).Error("application encryption " + phase + " failed")
		return
	}
	if job.State == "auditing" && stateBefore != "auditing" {
		entry.WithField("storage_location", job.StorageLocation).Info("application encryption audit started")
		return
	}
	if job.State == "completed" {
		entry.WithField("audit_result", "passed").Info("application encryption migration completed")
		return
	}
	if job.Cursor != "" {
		entry.WithField("cursor", job.Cursor).Info("application encryption " + phase + " batch committed")
	}
}
