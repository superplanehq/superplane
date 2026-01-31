package workers

import (
	"context"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type InstallationCleanupWorker struct {
	baseURL   string
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
}

func NewInstallationCleanupWorker(registry *registry.Registry, encryptor crypto.Encryptor, baseURL string) *InstallationCleanupWorker {
	return &InstallationCleanupWorker{
		semaphore: semaphore.NewWeighted(25),
		registry:  registry,
		encryptor: encryptor,
		baseURL:   baseURL,
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

	w.log("Cleaning up app installation %s", installation.ID)
	impl, err := w.registry.GetIntegration(installation.AppName)
	if err != nil {
		return err
	}

	err = impl.Cleanup(core.IntegrationCleanupContext{
		Configuration:  installation.Configuration.Data(),
		BaseURL:        w.baseURL,
		OrganizationID: installation.OrganizationID.String(),
		InstallationID: installation.ID.String(),
		HTTP:           contexts.NewHTTPContext(w.registry.GetHTTPClient()),
		Integration:    contexts.NewIntegrationContext(tx, nil, installation, w.encryptor, w.registry),
		Logger:         logging.ForAppInstallation(*installation),
	})

	if err != nil {
		return err
	}

	w.log("Cleanup completed for app installation %s - deleting", installation.ID)
	return tx.Unscoped().Delete(installation).Error
}

func (w *InstallationCleanupWorker) log(format string, v ...any) {
	log.Printf("[InstallationCleanupWorker] "+format, v...)
}
