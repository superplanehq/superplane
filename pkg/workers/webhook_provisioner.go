package workers

import (
	"context"
	"errors"
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
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

const (
	webhookProvisionerReasonSetupError = "setup_error"
)

type WebhookProvisioner struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
	baseURL   string
}

func NewWebhookProvisioner(baseURL string, encryptor crypto.Encryptor, registry *registry.Registry) *WebhookProvisioner {
	return &WebhookProvisioner{
		registry:  registry,
		baseURL:   baseURL,
		encryptor: encryptor,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *WebhookProvisioner) Start(ctx context.Context) {
	// On startup, reset any webhooks stuck in "provisioning" state
	// from a previous crash back to "pending" so they get retried.
	if count, err := models.ResetStuckProvisioningWebhooks(); err != nil {
		w.log("Error resetting stuck provisioning webhooks: %v", err)
	} else if count > 0 {
		w.log("Reset %d stuck provisioning webhook(s) back to pending", count)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			webhooks, err := models.ListPendingWebhooks()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordWebhookProvisionerWorkerWebhooksCount(context.Background(), len(webhooks))

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

			telemetry.RecordWebhookProvisionerWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

// LockAndProcessWebhook processes a webhook in 3 phases to avoid holding
// a DB connection during potentially long-running external API calls:
//
//   - Phase 1 (short tx): Lock the webhook and set state to "provisioning"
//   - Phase 2 (no tx): Run the external handler.Setup() call
//   - Phase 3 (short tx): Set state to "ready" or handle errors
func (w *WebhookProvisioner) LockAndProcessWebhook(webhook models.Webhook) error {
	if webhook.AppInstallationID == nil {
		return w.handleNonIntegrationWebhook(webhook)
	}

	return w.handleIntegrationWebhook(webhook)
}

func (w *WebhookProvisioner) handleNonIntegrationWebhook(webhook models.Webhook) error {
	//
	// Non-integration webhooks don't need external calls — lock and mark ready.
	//

	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	appName := "internal"

	defer func() {
		telemetry.RecordWebhookProvisionerWorkerWebhookProcessing(
			context.Background(),
			time.Since(start),
			outcome,
			reason,
			appName,
		)
	}()
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockWebhook(tx, webhook.ID)
		if err != nil {
			return err
		}

		return r.Ready(tx)
	})

	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		w.log("Webhook %s already being processed - skipping", webhook.ID)
		outcome = executorOutcomeSkipped
		reason = executorReasonLocked
		return nil
	}

	outcome = executorOutcomeFailed
	reason = classifyProcessError(err)
	return err
}

func (w *WebhookProvisioner) handleIntegrationWebhook(webhook models.Webhook) error {
	//
	// Webhooks for integrations need to call the external handler.Setup()
	// method, so we need to lock and mark as provisioning in a short
	// transaction, run the external call outside any transaction, and
	// finalize the state based on the result.
	//

	start := time.Now()
	outcome := executorOutcomeSuccess
	reason := executorReasonNone
	appName := "unknown"
	defer func() {
		telemetry.RecordWebhookProvisionerWorkerWebhookProcessing(
			context.Background(),
			time.Since(start),
			outcome,
			reason,
			appName,
		)
	}()

	//
	// Phase 1: Lock and mark as provisioning in a short transaction.
	//
	lockedWebhook, err := w.lockAndMarkProvisioning(webhook)
	if err != nil {
		w.log("Error locking and marking webhook %s as provisioning: %v", webhook.ID, err)
		outcome = executorOutcomeFailed
		reason = executorReasonInternal
		return err
	}

	if lockedWebhook == nil {
		w.log("Webhook %s already being processed - skipping", webhook.ID)
		outcome = executorOutcomeSkipped
		reason = executorReasonLocked
		return nil
	}

	//
	// Phase 2: Run handler.Setup() outside any transaction
	//
	metadata, appName, setupErr := w.runIntegrationSetup(lockedWebhook)

	//
	// Phase 3: Finalize state based on the result
	//
	if setupErr != nil {
		w.log("Error running integration setup for webhook %s: %v", webhook.ID, setupErr)
		outcome = executorOutcomeFailed
		reason = webhookProvisionerReasonSetupError

		err := w.handleProvisioningError(lockedWebhook, setupErr)
		if err != nil {
			w.log("Error handling provisioning error for webhook %s: %v", webhook.ID, err)
			reason = executorReasonInternal
		}

		return err
	}

	if err := w.markReady(lockedWebhook, metadata); err != nil {
		w.log("Error marking webhook %s as ready: %v", webhook.ID, err)
		outcome = executorOutcomeFailed
		reason = executorReasonInternal
		return err
	}

	return nil
}

// lockAndMarkProvisioning acquires a row lock and transitions the webhook
// from "pending" to "provisioning". Returns nil if the row was already picked
// up by another worker.
func (w *WebhookProvisioner) lockAndMarkProvisioning(webhook models.Webhook) (*models.Webhook, error) {
	var locked *models.Webhook

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockWebhook(tx, webhook.ID)
		if err != nil {
			return err
		}

		if err := r.MarkProvisioning(tx); err != nil {
			return err
		}

		locked = r
		return nil
	})

	if err != nil {
		w.log("Webhook %s already being processed - skipping", webhook.ID)
		return nil, nil
	}

	return locked, nil
}

// runIntegrationSetup calls the external webhook handler outside any
// DB transaction so the connection is released back to the pool.
func (w *WebhookProvisioner) runIntegrationSetup(webhook *models.Webhook) (any, string, error) {
	db := database.Conn()

	instance, err := models.FindUnscopedIntegrationInTransaction(db, *webhook.AppInstallationID)
	if err != nil {
		return nil, "", err
	}

	handler, err := w.registry.GetWebhookHandler(instance.AppName)
	if err != nil {
		return nil, "", err
	}

	logging.ForIntegration(*instance).
		WithField("source", "webhook").
		Info("Calling integration webhook setup handler")

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        w.registry.HTTPContext(),
		Integration: contexts.NewIntegrationContext(db, nil, instance, w.encryptor, w.registry, nil),
		Webhook:     contexts.NewWebhookContext(db, webhook, w.encryptor, w.baseURL),
		Logger:      logging.ForIntegration(*instance),
	})

	return metadata, instance.AppName, err
}

// markReady transitions the webhook to "ready" state.
func (w *WebhookProvisioner) markReady(webhook *models.Webhook, metadata any) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if metadata != nil {
			return webhook.ReadyWithMetadata(tx, metadata)
		}
		return webhook.Ready(tx)
	})
}

// handleProvisioningError handles a failed Setup() by either incrementing
// the retry count or marking the webhook as failed.
func (w *WebhookProvisioner) handleProvisioningError(webhook *models.Webhook, originalErr error) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if webhook.HasExceededRetries() {
			w.log("Webhook %s has exceeded max retries (%d), marking as failed", webhook.ID, webhook.MaxRetries)
			if err := webhook.MarkFailed(tx); err != nil {
				w.log("Error marking webhook %s as failed: %v", webhook.ID, err)
				return err
			}
			return nil
		}

		// Reset state back to pending so it can be retried.
		if err := tx.Model(webhook).Update("state", models.WebhookStatePending).Error; err != nil {
			return err
		}

		if err := webhook.IncrementRetry(tx); err != nil {
			w.log("Error incrementing retry count for webhook %s: %v", webhook.ID, err)
			return err
		}

		w.log("Webhook %s provisioning failed (attempt %d/%d): %v", webhook.ID, webhook.RetryCount, webhook.MaxRetries, originalErr)
		return nil
	})
}

func (w *WebhookProvisioner) log(format string, v ...any) {
	log.Printf("[WebhookProvisioner] "+format, v...)
}
