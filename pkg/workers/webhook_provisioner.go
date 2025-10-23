package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

type WebhookProvisioner struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	baseURL   string
}

func NewWebhookProvisioner(baseURL string, registry *registry.Registry) *WebhookProvisioner {
	return &WebhookProvisioner{
		registry:  registry,
		baseURL:   baseURL,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *WebhookProvisioner) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			webhooks, err := models.ListPendingWebhooks()
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

func (w *WebhookProvisioner) LockAndProcessWebhook(webhook models.Webhook) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockWebhook(tx, webhook.ID)
		if err != nil {
			w.log("Webhook %s already being processed - skipping", webhook.ID)
			return nil
		}

		return w.processWebhook(tx, r)
	})
}

func (w *WebhookProvisioner) processWebhook(tx *gorm.DB, webhook *models.Webhook) error {
	if webhook.IntegrationID == nil {
		return webhook.Ready(tx)
	}

	integration, err := models.FindIntegrationByIDInTransaction(tx, *webhook.IntegrationID)
	if err != nil {
		return err
	}

	resourceManager, err := w.registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		return err
	}

	webhookResource := webhook.Resource.Data()
	resource, err := resourceManager.Get(webhookResource.Type, webhookResource.ID)
	if err != nil {
		return err
	}

	webhookMetadata, err := resourceManager.SetupWebhookV2(integrations.WebhookOptionsV2{
		Resource:      resource,
		Configuration: webhook.Configuration,
		URL:           fmt.Sprintf("%s/api/v1/webhooks/%s", w.baseURL, webhook.ID.String()),
		Secret:        webhook.Secret,
	})

	if err != nil {
		return err
	}

	return webhook.ReadyWithMetadata(tx, webhookMetadata)
}

func (w *WebhookProvisioner) log(format string, v ...any) {
	log.Printf("[WebhookProvisioner] "+format, v...)
}
