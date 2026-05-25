package workers

import (
	"fmt"
	"log"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookLegacyMigrationJob migrates legacy-mode webhook registrations to the ops path.
// For each legacy webhook whose handler implements ScopeKeyer it:
//   - Derives the scope key and config hash from the current configuration.
//   - Sets scope_key and config_hash on the webhook row.
//   - Flips provisioning_mode to 'ops'.
//   - Enqueues a 'create' operation when the webhook is not already in 'ready' state,
//     so the ops provisioner (re-)provisions it through the new path.
//
// Handlers that have not yet adopted ScopeKeyer are left in legacy mode and continue
// to be serviced by the legacy WebhookProvisioner until they migrate.
//
// Pre-condition: the Phase 3 dedupe job must have run and the unique index
// 20260525140000_add-webhook-unique-scope must be applied, so scope_key conflicts
// are caught by the DB constraint rather than silently creating duplicates.
//
// Run via: RUN_WEBHOOK_LEGACY_MIGRATION_JOB=yes
type WebhookLegacyMigrationJob struct {
	registry *registry.Registry
}

func NewWebhookLegacyMigrationJob(r *registry.Registry) *WebhookLegacyMigrationJob {
	return &WebhookLegacyMigrationJob{registry: r}
}

func (j *WebhookLegacyMigrationJob) Run() error {
	db := database.Conn()

	var webhooks []models.Webhook
	if err := db.
		Where("provisioning_mode = ?", models.WebhookProvisioningModeLegacy).
		Find(&webhooks).Error; err != nil {
		return fmt.Errorf("listing legacy webhooks: %w", err)
	}

	j.log("Found %d legacy-mode registration(s)", len(webhooks))

	migrated, skipped, failed := 0, 0, 0
	for _, w := range webhooks {
		ok, err := j.migrate(db, w)
		if err != nil {
			j.log("Error migrating webhook %s (app_installation=%v): %v", w.ID, w.AppInstallationID, err)
			failed++
			continue
		}
		if ok {
			migrated++
		} else {
			skipped++
		}
	}

	j.log("Migration complete: migrated=%d skipped=%d failed=%d", migrated, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d webhook(s) failed to migrate", failed)
	}
	return nil
}

// migrate attempts to migrate one legacy-mode webhook to ops-mode.
// Returns (true, nil) on success, (false, nil) when skipped.
func (j *WebhookLegacyMigrationJob) migrate(db *gorm.DB, w models.Webhook) (bool, error) {
	// Skip webhooks already being deleted and those with no installation reference.
	if w.State == models.WebhookStateDeletingPending || w.AppInstallationID == nil {
		return false, nil
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(db, *w.AppInstallationID)
	if err != nil {
		return false, fmt.Errorf("loading integration: %w", err)
	}

	handler, err := j.registry.GetWebhookHandler(integration.AppName)
	if err != nil {
		return false, fmt.Errorf("getting webhook handler for %s: %w", integration.AppName, err)
	}

	sk, ok := handler.(core.ScopeKeyer)
	if !ok {
		// Handler has not adopted ScopeKeyer yet; leave in legacy mode.
		return false, nil
	}

	config := w.Configuration.Data()

	scopeKey, err := sk.ScopeKey(config)
	if err != nil {
		return false, fmt.Errorf("deriving scope key: %w", err)
	}

	configHash, err := opConfigHash(config)
	if err != nil {
		return false, fmt.Errorf("hashing config: %w", err)
	}

	return true, db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Provisioning state means the process crashed mid-provision; treat as pending.
		newState := w.State
		if newState == models.WebhookStateProvisioning {
			newState = models.WebhookStatePending
		}

		if err := tx.Model(&w).Updates(map[string]any{
			"scope_key":         scopeKey,
			"config_hash":       configHash,
			"provisioning_mode": models.WebhookProvisioningModeOps,
			"state":             newState,
			"updated_at":        now,
		}).Error; err != nil {
			return fmt.Errorf("updating webhook: %w", err)
		}

		// Ready webhooks are already provisioned; no operation needed.
		if w.State == models.WebhookStateReady {
			return nil
		}

		iKey := opIdempotencyKey(w.ID, models.WebhookOperationTypeCreate, configHash, w.SecretVersion)
		op := models.WebhookOperation{
			WebhookID:         w.ID,
			OperationType:     models.WebhookOperationTypeCreate,
			DesiredConfig:     datatypes.NewJSONType(config),
			DesiredConfigHash: &configHash,
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

func (j *WebhookLegacyMigrationJob) log(format string, v ...any) {
	log.Printf("[WebhookLegacyMigrationJob] "+format, v...)
}
