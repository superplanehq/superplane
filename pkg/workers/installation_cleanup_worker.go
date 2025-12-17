package workers

import (
	"context"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type InstallationCleanupWorker struct {
	semaphore *semaphore.Weighted
}

func NewInstallationCleanupWorker() *InstallationCleanupWorker {
	return &InstallationCleanupWorker{
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *InstallationCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			installations, err := models.ListDeletedAppInstallations()
			if err != nil {
				w.log("Error finding deleted app installations: %v", err)
			}

			for _, installation := range installations {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(installation models.AppInstallation) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessInstallation(installation); err != nil {
						w.log("Error processing app installation %s: %v", installation.ID, err)
					}
				}(installation)
			}
		}
	}
}

func (w *InstallationCleanupWorker) LockAndProcessInstallation(installation models.AppInstallation) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockAppInstallation(tx, installation.ID)
		if err != nil {
			w.log("App installation %s already being processed - skipping", installation.ID)
			return nil
		}

		w.log("Processing app installation %s", installation.ID)
		return w.processAppInstallation(tx, r)
	})
}

func (w *InstallationCleanupWorker) processAppInstallation(tx *gorm.DB, installation *models.AppInstallation) error {
	webhooks, err := models.ListUnscopedAppInstallationWebhooks(tx, installation.ID)
	if err != nil {
		return err
	}

	if len(webhooks) > 0 {
		w.log("App installation %s still has %d webhooks - skipping", installation.ID, len(webhooks))
		return nil
	}

	w.log("Deleting app installation %s", installation.ID)
	return tx.Unscoped().Delete(installation).Error
}

func (w *InstallationCleanupWorker) log(format string, v ...any) {
	log.Printf("[InstallationCleanupWorker] "+format, v...)
}
