package workers

import (
	"context"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WebhookCleanupWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
	baseURL   string
	logger    *log.Entry
}

func NewWebhookCleanupWorker(encryptor crypto.Encryptor, registry *registry.Registry, baseURL string) *WebhookCleanupWorker {
	return &WebhookCleanupWorker{
		registry:  registry,
		encryptor: encryptor,
		semaphore: semaphore.NewWeighted(25),
		baseURL:   baseURL,
		logger:    log.WithFields(log.Fields{"worker": "WebhookCleanupWorker"}),
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
			tickStart := time.Now()

			webhooks, err := models.ListDeletedWebhooks()
			if err != nil {
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordWebhookCleanupWorkerWebhooksCount(context.Background(), len(webhooks))

			for _, webhook := range webhooks {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(webhook models.Webhook) {
					defer w.semaphore.Release(1)

					logger := logging.WithWebhook(w.logger, webhook)
					if err := w.LockAndProcessWebhook(logger, webhook); err != nil {
						logger.Errorf("Error processing webhook: %v", err)
					}
				}(webhook)
			}

			telemetry.RecordWebhookCleanupWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *WebhookCleanupWorker) LockAndProcessWebhook(logger *log.Entry, webhook models.Webhook) error {
	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	defer func() {
		telemetry.RecordWebhookCleanupWorkerWebhookProcessing(
			context.Background(),
			time.Since(start),
			outcome,
			reason,
		)
	}()

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockDeletedWebhook(tx, webhook.ID)
		if err != nil {
			logger.Info("Webhook already being processed - skipping")
			outcome = executorOutcomeSkipped
			reason = executorReasonLocked
			return nil
		}

		logger.Info("Processing webhook")
		return w.processWebhook(tx, logger, r)
	})
	if err != nil {
		logger.Errorf("Error processing webhook: %v", err)
		outcome = executorOutcomeFailed
		reason = classifyProcessError(err)
	}

	return err
}

func (w *WebhookCleanupWorker) processWebhook(tx *gorm.DB, logger *log.Entry, webhook *models.Webhook) error {
	if webhook.AppInstallationID != nil {
		return w.processAppInstallationWebhook(tx, logger, webhook)
	}

	return tx.Unscoped().Delete(webhook).Error
}

func (w *WebhookCleanupWorker) processAppInstallationWebhook(tx *gorm.DB, logger *log.Entry, webhook *models.Webhook) error {
	instance, err := models.FindMaybeDeletedIntegrationInTransaction(tx, *webhook.AppInstallationID)
	if err != nil {
		return err
	}

	handler, err := w.registry.GetWebhookHandler(instance.AppName)
	if err != nil {
		return err
	}

	err = handler.Cleanup(core.WebhookHandlerContext{
		HTTP:        w.registry.HTTPContextInTransaction(tx),
		Integration: contexts.NewIntegrationContext(tx, nil, instance, w.encryptor, w.registry, nil),
		Webhook:     contexts.NewWebhookContext(tx, webhook, w.encryptor, w.baseURL),
		Logger:      logging.WithIntegration(logger, *instance),
	})

	if err != nil {
		logger.Errorf("Best-effort cleanup failed for webhook: %v", err)
	}

	return tx.Unscoped().Delete(webhook).Error
}
