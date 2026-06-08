package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

// WebhookReconciler is the Phase 2 reconciler. It groups active
// WebhookSubscriptionBindings by (app_installation_id, scope_key), computes the
// deterministic desired merged config, and — when FeatureWebhookReconciler is enabled
// for the registration's org — creates/updates/deletes webhook registrations and
// enqueues webhook_operations for the ops provisioner to execute.
//
// When FeatureWebhookReconciler is disabled it falls back to shadow/audit mode:
// it logs drift but does not mutate any state.
type WebhookReconciler struct {
	registry  *registry.Registry
	encryptor crypto.Encryptor
	baseURL   string
	interval  time.Duration
}

func NewWebhookReconciler(registry *registry.Registry, encryptor crypto.Encryptor, baseURL string) *WebhookReconciler {
	return &WebhookReconciler{
		registry:  registry,
		encryptor: encryptor,
		baseURL:   baseURL,
		interval:  time.Minute,
	}
}

func (r *WebhookReconciler) Start(ctx context.Context) {
	// On startup reset stuck running operations so they are retried.
	if count, err := models.ResetStuckRunningOperations(); err != nil {
		r.log("Error resetting stuck running operations: %v", err)
	} else if count > 0 {
		r.log("Reset %d stuck running operation(s) back to queued", count)
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reconcileAll()
		}
	}
}

func (r *WebhookReconciler) reconcileAll() {
	groups, err := models.ListActiveBindingGroups()
	if err != nil {
		r.log("Error listing binding groups: %v", err)
		return
	}

	for _, group := range groups {
		if err := r.reconcileGroup(group); err != nil {
			r.log("Error reconciling group app_installation=%s scope=%q: %v",
				group.AppInstallationID, group.ScopeKey, err)
		}
	}

	r.reconcileOrphans()
}

// reconcileGroup handles one (app_installation_id, scope_key) pair.
func (r *WebhookReconciler) reconcileGroup(group models.BindingGroup) error {
	db := database.Conn()

	bindings, err := models.ListActiveBindingsForGroup(group.AppInstallationID, group.ScopeKey)
	if err != nil {
		return fmt.Errorf("listing bindings: %w", err)
	}
	if len(bindings) == 0 {
		return nil
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(db, group.AppInstallationID)
	if err != nil {
		return fmt.Errorf("loading integration: %w", err)
	}

	handler, err := r.registry.GetWebhookHandler(integration.AppName)
	if err != nil {
		return fmt.Errorf("getting webhook handler: %w", err)
	}

	// Deterministic desired config: stable fold over bindings sorted by id ASC.
	desired := bindings[0].RequestedConfig.Data()
	for _, b := range bindings[1:] {
		merged, _, mergeErr := handler.Merge(desired, b.RequestedConfig.Data())
		if mergeErr != nil {
			return fmt.Errorf("merging configs (binding %s): %w", b.ID, mergeErr)
		}
		desired = merged
	}

	desiredHash, err := opConfigHash(desired)
	if err != nil {
		return fmt.Errorf("hashing desired config: %w", err)
	}

	// Check whether the reconciler is allowed to mutate state for this org.
	active, err := models.HasExperimentalFeature(integration.OrganizationID, features.FeatureWebhookReconciler)
	if err != nil {
		return fmt.Errorf("checking feature flag: %w", err)
	}

	// Find current registration.
	current, err := models.FindWebhookByScope(db, group.AppInstallationID, group.ScopeKey)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("finding registration: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No registration exists.
		if !active {
			r.log("Shadow: no registration for app_installation=%s scope=%q — would create",
				group.AppInstallationID, group.ScopeKey)
			return nil
		}
		return r.createRegistrationAndOp(db, integration, group.ScopeKey, desired, desiredHash)
	}

	// Registration exists — check for drift.
	matches, err := handler.CompareConfig(current.Configuration.Data(), desired)
	if err != nil {
		return fmt.Errorf("comparing configs: %w", err)
	}

	if matches {
		return nil
	}

	if !active {
		r.log("Shadow: drift detected for app_installation=%s scope=%q webhook=%s",
			group.AppInstallationID, group.ScopeKey, current.ID)
		return nil
	}

	return r.enqueueUpdateOp(db, current, desired, desiredHash)
}

// reconcileOrphans finds ops-mode registrations with no active bindings and enqueues
// delete operations for them.
func (r *WebhookReconciler) reconcileOrphans() {
	orphans, err := models.ListOrphanedOpsWebhooks()
	if err != nil {
		r.log("Error listing orphaned registrations: %v", err)
		return
	}

	for _, w := range orphans {
		if err := r.enqueueDeleteOp(database.Conn(), &w); err != nil {
			r.log("Error enqueuing delete op for webhook %s: %v", w.ID, err)
		}
	}
}

// createRegistrationAndOp creates a new ops-mode webhook registration and enqueues
// a 'create' operation for the ops provisioner to execute. Both writes are in one
// transaction so a partial failure leaves no phantom registration.
func (r *WebhookReconciler) createRegistrationAndOp(
	db *gorm.DB,
	integration *models.Integration,
	scopeKey string,
	desiredConfig any,
	desiredHash string,
) error {
	webhookID := uuid.New()
	_, encryptedKey, err := crypto.NewRandomKey(context.Background(), r.encryptor, webhookID.String())
	if err != nil {
		return fmt.Errorf("generating webhook secret: %w", err)
	}

	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		webhook := models.Webhook{
			ID:                webhookID,
			State:             models.WebhookStatePending,
			Secret:            encryptedKey,
			Configuration:     datatypes.NewJSONType(desiredConfig),
			AppInstallationID: &integration.ID,
			ScopeKey:          &scopeKey,
			ConfigHash:        &desiredHash,
			SecretVersion:     1,
			ProvisioningMode:  models.WebhookProvisioningModeOps,
			CreatedAt:         &now,
			UpdatedAt:         &now,
		}
		if err := tx.Create(&webhook).Error; err != nil {
			return fmt.Errorf("creating registration: %w", err)
		}

		iKey := opIdempotencyKey(webhookID, models.WebhookOperationTypeCreate, desiredHash, 1)
		op := models.WebhookOperation{
			WebhookID:         webhookID,
			OperationType:     models.WebhookOperationTypeCreate,
			DesiredConfig:     datatypes.NewJSONType(desiredConfig),
			DesiredConfigHash: &desiredHash,
			IdempotencyKey:    iKey,
			State:             models.WebhookOperationStateQueued,
			MaxAttempts:       5,
			NextAttemptAt:     now,
			CreatedAt:         &now,
			UpdatedAt:         &now,
		}
		return models.EnqueueWebhookOperation(tx, &op)
	})
}

// enqueueUpdateOp updates the registration's desired config and enqueues an 'update'
// operation. The config is written to the webhook row first so the ops provisioner
// sees the new config via WebhookContext.GetConfiguration().
func (r *WebhookReconciler) enqueueUpdateOp(
	db *gorm.DB,
	webhook *models.Webhook,
	desiredConfig any,
	desiredHash string,
) error {
	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(webhook).Updates(map[string]any{
			"configuration": datatypes.NewJSONType(desiredConfig),
			"config_hash":   desiredHash,
			"state":         models.WebhookStatePending,
			"updated_at":    now,
		}).Error; err != nil {
			return fmt.Errorf("updating registration config: %w", err)
		}

		iKey := opIdempotencyKey(webhook.ID, models.WebhookOperationTypeUpdate, desiredHash, webhook.SecretVersion)
		op := models.WebhookOperation{
			WebhookID:         webhook.ID,
			OperationType:     models.WebhookOperationTypeUpdate,
			DesiredConfig:     datatypes.NewJSONType(desiredConfig),
			DesiredConfigHash: &desiredHash,
			IdempotencyKey:    iKey,
			State:             models.WebhookOperationStateQueued,
			MaxAttempts:       5,
			NextAttemptAt:     now,
			CreatedAt:         &now,
			UpdatedAt:         &now,
		}
		return models.EnqueueWebhookOperation(tx, &op)
	})
}

// enqueueDeleteOp marks the registration as deleting and enqueues a 'delete' operation.
func (r *WebhookReconciler) enqueueDeleteOp(db *gorm.DB, webhook *models.Webhook) error {
	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(webhook).Updates(map[string]any{
			"state":      models.WebhookStateDeletingPending,
			"updated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("marking registration as deleting: %w", err)
		}

		hashVal := ""
		if webhook.ConfigHash != nil {
			hashVal = *webhook.ConfigHash
		}
		iKey := opIdempotencyKey(webhook.ID, models.WebhookOperationTypeDelete, hashVal, webhook.SecretVersion)
		op := models.WebhookOperation{
			WebhookID:      webhook.ID,
			OperationType:  models.WebhookOperationTypeDelete,
			IdempotencyKey: iKey,
			State:          models.WebhookOperationStateQueued,
			MaxAttempts:    5,
			NextAttemptAt:  now,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		return models.EnqueueWebhookOperation(tx, &op)
	})
}

// opIdempotencyKey derives a stable idempotency key for a webhook operation.
func opIdempotencyKey(webhookID uuid.UUID, opType, configHash string, secretVersion int) string {
	input := fmt.Sprintf("%s:%s:%s:%d", webhookID, opType, configHash, secretVersion)
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

// opConfigHash returns the hex-encoded SHA-256 of the JSON-marshaled config.
// encoding/json marshals map keys in sorted order so this is deterministic.
func opConfigHash(config any) (string, error) {
	b, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (r *WebhookReconciler) log(format string, v ...any) {
	log.Printf("[WebhookReconciler] "+format, v...)
}
