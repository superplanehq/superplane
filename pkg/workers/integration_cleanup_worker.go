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

type IntegrationCleanupWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
	baseURL   string
}

func NewIntegrationCleanupWorker(registry *registry.Registry, encryptor crypto.Encryptor, baseURL string) *IntegrationCleanupWorker {
	return &IntegrationCleanupWorker{
		semaphore: semaphore.NewWeighted(25),
		registry:  registry,
		encryptor: encryptor,
		baseURL:   baseURL,
	}
}

func (w *IntegrationCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			integrations, err := models.ListDeletedIntegrations()
			if err != nil {
				w.log("Error finding deleted integrations: %v", err)
			}

			for _, integration := range integrations {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(integration models.Integration) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessIntegration(integration); err != nil {
						w.log("Error processing integration %s: %v", integration.ID, err)
					}
				}(integration)
			}
		}
	}
}

func (w *IntegrationCleanupWorker) LockAndProcessIntegration(integration models.Integration) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockIntegration(tx, integration.ID)
		if err != nil {
			w.log("Integration %s already being processed - skipping", integration.ID)
			return nil
		}

		w.log("Processing integration %s", integration.ID)
		return w.processIntegration(tx, r)
	})
}

func (w *IntegrationCleanupWorker) processIntegration(tx *gorm.DB, integration *models.Integration) error {
	webhooks, err := models.ListUnscopedIntegrationWebhooks(tx, integration.ID)
	if err != nil {
		return err
	}

	if len(webhooks) > 0 {
		w.log("Integration %s still has %d webhooks - skipping", integration.ID, len(webhooks))
		return nil
	}

	w.log("Cleaning up app installation %s", integration.ID)
	impl, err := w.registry.GetIntegration(integration.AppName)
	if err != nil {
		return err
	}

	err = impl.Cleanup(core.IntegrationCleanupContext{
		Configuration:  integration.Configuration.Data(),
		BaseURL:        w.baseURL,
		OrganizationID: integration.OrganizationID.String(),
		InstallationID: integration.ID.String(),
		HTTP:           contexts.NewHTTPContext(w.registry.GetHTTPClient()),
		Integration:    contexts.NewIntegrationContext(tx, nil, integration, w.encryptor, w.registry),
		Logger:         logging.ForIntegration(*integration),
	})

	if err != nil {
		return err
	}

	w.log("Cleanup completed for integration %s - deleting", integration.ID)
	return tx.Unscoped().Delete(integration).Error
}

func (w *IntegrationCleanupWorker) log(format string, v ...any) {
	log.Printf("[IntegrationCleanupWorker] "+format, v...)
}
