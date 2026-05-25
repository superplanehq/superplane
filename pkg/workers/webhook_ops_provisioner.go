package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// WebhookOpsProvisioner executes webhook_operations enqueued by the WebhookReconciler.
// It follows the same 3-phase pattern as WebhookProvisioner to avoid holding a DB
// connection during potentially slow external provider calls.
type WebhookOpsProvisioner struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
	baseURL   string
}

func NewWebhookOpsProvisioner(baseURL string, encryptor crypto.Encryptor, registry *registry.Registry) *WebhookOpsProvisioner {
	return &WebhookOpsProvisioner{
		registry:  registry,
		encryptor: encryptor,
		baseURL:   baseURL,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *WebhookOpsProvisioner) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ops, err := models.ListQueuedOperations()
			if err != nil {
				w.log("Error listing queued operations: %v", err)
				continue
			}

			for _, op := range ops {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(op models.WebhookOperation) {
					defer w.semaphore.Release(1)
					if err := w.lockAndProcess(op); err != nil {
						w.log("Error processing operation %s: %v", op.ID, err)
					}
				}(op)
			}
		}
	}
}

// lockAndProcess is the 3-phase driver: lock, call provider, persist outcome.
func (w *WebhookOpsProvisioner) lockAndProcess(op models.WebhookOperation) error {
	// Phase 1: acquire row lock and transition to 'running'.
	var locked *models.WebhookOperation
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		row, err := models.LockOperation(tx, op.ID)
		if err != nil {
			return err
		}
		locked = row
		return tx.Model(locked).Updates(map[string]any{
			"state":      models.WebhookOperationStateRunning,
			"updated_at": time.Now(),
		}).Error
	})
	if err != nil {
		// Row already claimed or no longer queued — skip silently.
		return nil //nolint:nilerr
	}

	// Phase 2: execute the provider call outside any transaction.
	metadata, providerErr := w.callProvider(locked)

	// Phase 3: persist outcome in a short transaction.
	if providerErr != nil {
		return w.handleFailure(locked, providerErr)
	}
	return w.handleSuccess(locked, metadata)
}

// callProvider loads the registration and integration, then dispatches to the
// correct handler method. Returns provider-supplied metadata on success.
func (w *WebhookOpsProvisioner) callProvider(op *models.WebhookOperation) (any, error) {
	db := database.Conn()

	webhook, err := models.FindWebhookInTransaction(db, op.WebhookID)
	if err != nil {
		return nil, fmt.Errorf("loading registration: %w", err)
	}

	if webhook.AppInstallationID == nil {
		return nil, fmt.Errorf("registration %s has no app_installation_id", webhook.ID)
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(db, *webhook.AppInstallationID)
	if err != nil {
		return nil, fmt.Errorf("loading integration: %w", err)
	}

	handler, err := w.registry.GetWebhookHandler(integration.AppName)
	if err != nil {
		return nil, fmt.Errorf("getting webhook handler: %w", err)
	}

	hCtx := core.WebhookHandlerContext{
		HTTP:        w.registry.HTTPContext(),
		Integration: contexts.NewIntegrationContext(db, nil, integration, w.encryptor, w.registry, nil),
		Webhook:     contexts.NewWebhookContext(db, webhook, w.encryptor, w.baseURL),
		Logger:      logging.ForIntegration(*integration),
	}

	switch op.OperationType {
	case models.WebhookOperationTypeCreate, models.WebhookOperationTypeUpdate:
		// webhook.Configuration already holds the desired config (set by the reconciler
		// before enqueuing the operation), so handler.Setup sees the correct config.
		return handler.Setup(hCtx)

	case models.WebhookOperationTypeDelete:
		return nil, handler.Cleanup(hCtx)

	default:
		return nil, fmt.Errorf("unsupported operation type: %s", op.OperationType)
	}
}

// handleSuccess writes all outcome state in a single transaction.
func (w *WebhookOpsProvisioner) handleSuccess(op *models.WebhookOperation, metadata any) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		if err := tx.Model(op).Updates(map[string]any{
			"state":      models.WebhookOperationStateSucceeded,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		if op.OperationType == models.WebhookOperationTypeDelete {
			// Soft-delete the registration now that the provider confirmed removal.
			return tx.Where("id = ?", op.WebhookID).Delete(&models.Webhook{}).Error
		}

		updates := map[string]any{
			"state":               models.WebhookStateReady,
			"last_provisioned_at": now,
			"updated_at":          now,
		}
		if metadata != nil {
			updates["metadata"] = datatypes.NewJSONType(metadata)
		}
		return tx.Model(&models.Webhook{}).
			Where("id = ?", op.WebhookID).
			Updates(updates).Error
	})
}

// handleFailure applies exponential backoff or marks the operation terminally failed.
func (w *WebhookOpsProvisioner) handleFailure(op *models.WebhookOperation, providerErr error) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		op.AttemptCount++
		errMsg := providerErr.Error()

		if op.AttemptCount >= op.MaxAttempts {
			w.log("Operation %s (%s) reached max attempts (%d): %v",
				op.ID, op.OperationType, op.MaxAttempts, providerErr)

			return tx.Model(op).Updates(map[string]any{
				"state":              models.WebhookOperationStateFailedTerminal,
				"attempt_count":      op.AttemptCount,
				"last_error_message": errMsg,
				"last_error_at":      now,
				"updated_at":         now,
			}).Error
		}

		// Exponential backoff capped at 10 minutes: 10s, 20s, 40s, 80s …
		backoff := time.Duration(10<<uint(op.AttemptCount-1)) * time.Second
		if backoff > 10*time.Minute {
			backoff = 10 * time.Minute
		}
		nextAttempt := now.Add(backoff)

		w.log("Operation %s (%s) failed (attempt %d/%d), retry at %s: %v",
			op.ID, op.OperationType, op.AttemptCount, op.MaxAttempts,
			nextAttempt.Format(time.RFC3339), providerErr)

		return tx.Model(op).Updates(map[string]any{
			"state":              models.WebhookOperationStateFailedRetryable,
			"attempt_count":      op.AttemptCount,
			"next_attempt_at":    nextAttempt,
			"last_error_message": errMsg,
			"last_error_at":      now,
			"updated_at":         now,
		}).Error
	})
}

func (w *WebhookOpsProvisioner) log(format string, v ...any) {
	log.Printf("[WebhookOpsProvisioner] "+format, v...)
}
