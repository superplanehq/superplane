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
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WebhookCleanupWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
	baseURL   string
}

func NewWebhookCleanupWorker(encryptor crypto.Encryptor, registry *registry.Registry, baseURL string) *WebhookCleanupWorker {
	return &WebhookCleanupWorker{
		registry:  registry,
		encryptor: encryptor,
		semaphore: semaphore.NewWeighted(25),
		baseURL:   baseURL,
	}
}

func (w *WebhookCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			webhooks, err := models.ListDeletedWebhooks()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, webhook := range webhooks {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(webhook models.Webhook) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessWebhook(webhook); err != nil {
						w.log("Error processing webhook %s: %v", webhook.ID, err)
					}
				}(webhook)
			}
		}
	}
}

func (w *WebhookCleanupWorker) LockAndProcessWebhook(webhook models.Webhook) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockWebhook(tx, webhook.ID)
		if err != nil {
			w.log("Webhook %s already being processed - skipping", webhook.ID)
			return nil
		}

		w.log("Processing webhook %s", webhook.ID)
		return w.processWebhook(tx, r)
	})
}

func (w *WebhookCleanupWorker) processWebhook(tx *gorm.DB, webhook *models.Webhook) error {
	if webhook.AppInstallationID != nil {
		return w.processAppInstallationWebhook(tx, webhook)
	}

	if webhook.IntegrationID != nil {
		return w.processIntegrationWebhook(tx, webhook)
	}

	return tx.Unscoped().Delete(webhook).Error
}

func (w *WebhookCleanupWorker) processAppInstallationWebhook(tx *gorm.DB, webhook *models.Webhook) error {
	appInstallation, err := models.FindMaybeDeletedInstallationInTransaction(tx, *webhook.AppInstallationID)
	if err != nil {
		return err
	}

	app, err := w.registry.GetApplication(appInstallation.AppName)
	if err != nil {
		return err
	}

	err = app.CleanupWebhook(core.CleanupWebhookContext{
		Webhook:         contexts.NewWebhookContext(tx, webhook, w.encryptor, w.baseURL),
		AppInstallation: contexts.NewAppInstallationContext(tx, nil, appInstallation, w.encryptor, w.registry),
	})

	if err != nil {
		return err
	}

	return tx.Unscoped().Delete(webhook).Error
}

func (w *WebhookCleanupWorker) processIntegrationWebhook(tx *gorm.DB, webhook *models.Webhook) error {
	integration, err := models.FindIntegrationByIDInTransaction(tx, *webhook.IntegrationID)
	if err != nil {
		return err
	}

	resourceManager, err := w.registry.NewResourceManagerInTransaction(context.Background(), tx, integration)
	if err != nil {
		return err
	}

	webhookResource := webhook.Resource.Data()
	resource, err := resourceManager.Get(webhookResource.Type, webhookResource.ID)
	if err != nil {
		return err
	}

	err = resourceManager.CleanupWebhook(integrations.WebhookOptions{
		Resource:      resource,
		Configuration: webhook.Configuration.Data(),
		Metadata:      webhook.Metadata.Data(),
	})

	if err != nil {
		return err
	}

	return tx.Unscoped().Delete(webhook).Error
}

func (w *WebhookCleanupWorker) log(format string, v ...any) {
	log.Printf("[WebhookCleanupWorker] "+format, v...)
}
