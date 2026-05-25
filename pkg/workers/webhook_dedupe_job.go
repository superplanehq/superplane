package workers

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// WebhookDedupeJob is a one-shot admin job that collapses duplicate webhook
// registrations sharing the same (app_installation_id, scope_key).
//
// For each duplicate group the job:
//  1. Picks the canonical registration (oldest in 'ready' state, or oldest overall).
//  2. Rebinds all workflow_nodes and subscription bindings pointing to non-canonical
//     registrations to the canonical one.
//  3. Marks non-canonical registrations as provisioning_mode='ops' so the reconciler's
//     orphan detection enqueues delete operations for them.
//
// This job must complete with zero failures before applying the unique index migration
// 20260525140000_add-webhook-unique-scope.
//
// Run via: RUN_WEBHOOK_DEDUPE_JOB=yes
type WebhookDedupeJob struct{}

func NewWebhookDedupeJob() *WebhookDedupeJob { return &WebhookDedupeJob{} }

type duplicateGroup struct {
	AppInstallationID uuid.UUID
	ScopeKey          string
	Count             int64
}

func (j *WebhookDedupeJob) Run() error {
	db := database.Conn()

	var groups []duplicateGroup
	if err := db.Model(&models.Webhook{}).
		Select("app_installation_id, scope_key, COUNT(*) AS count").
		Where("scope_key IS NOT NULL").
		Group("app_installation_id, scope_key").
		Having("COUNT(*) > 1").
		Scan(&groups).Error; err != nil {
		return fmt.Errorf("finding duplicate groups: %w", err)
	}

	if len(groups) == 0 {
		j.log("No duplicate scope registrations found — safe to apply unique index migration")
		return nil
	}

	j.log("Found %d duplicate group(s) to collapse", len(groups))

	deduped, failed := 0, 0
	for _, g := range groups {
		if err := j.dedupeGroup(db, g); err != nil {
			j.log("Error deduping group app_installation=%s scope=%q: %v",
				g.AppInstallationID, g.ScopeKey, err)
			failed++
			continue
		}
		deduped++
	}

	j.log("Dedupe complete: collapsed=%d failed=%d", deduped, failed)

	if failed > 0 {
		return fmt.Errorf("%d group(s) failed to dedupe", failed)
	}
	return nil
}

func (j *WebhookDedupeJob) dedupeGroup(db *gorm.DB, g duplicateGroup) error {
	var webhooks []models.Webhook
	if err := db.
		Where("app_installation_id = ? AND scope_key = ?", g.AppInstallationID, g.ScopeKey).
		Order("created_at ASC").
		Find(&webhooks).Error; err != nil {
		return fmt.Errorf("loading duplicate registrations: %w", err)
	}

	if len(webhooks) <= 1 {
		return nil
	}

	canonical := j.pickCanonical(webhooks)
	j.log("Group app_installation=%s scope=%q: canonical=%s (%d total)",
		g.AppInstallationID, g.ScopeKey, canonical.ID, len(webhooks))

	return db.Transaction(func(tx *gorm.DB) error {
		for _, w := range webhooks {
			if w.ID == canonical.ID {
				continue
			}
			if err := j.rebindAndMark(tx, w.ID, canonical.ID); err != nil {
				return fmt.Errorf("rebinding webhook %s: %w", w.ID, err)
			}
		}
		return nil
	})
}

// pickCanonical returns the registration to keep: the oldest in 'ready' state,
// or the oldest overall when none are ready.
func (j *WebhookDedupeJob) pickCanonical(webhooks []models.Webhook) models.Webhook {
	for _, w := range webhooks { // already ordered by created_at ASC
		if w.State == models.WebhookStateReady {
			return w
		}
	}
	return webhooks[0]
}

// rebindAndMark reassigns all references from the duplicate to the canonical registration,
// then marks the duplicate as provisioning_mode='ops' so the reconciler's orphan detection
// enqueues a delete operation for it.
func (j *WebhookDedupeJob) rebindAndMark(tx *gorm.DB, duplicateID, canonicalID uuid.UUID) error {
	now := time.Now()

	if err := tx.Model(&models.CanvasNode{}).
		Where("webhook_id = ?", duplicateID).
		Updates(map[string]any{
			"webhook_id": canonicalID,
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("rebinding workflow_nodes: %w", err)
	}

	if err := tx.Model(&models.WebhookSubscriptionBinding{}).
		Where("webhook_id = ? AND active = true", duplicateID).
		Updates(map[string]any{
			"webhook_id": canonicalID,
			"updated_at": now,
		}).Error; err != nil {
		return fmt.Errorf("rebinding subscription bindings: %w", err)
	}

	// Marking as ops-mode means ListOrphanedOpsWebhooks will pick it up once all
	// bindings have been rebased to the canonical registration (i.e., none remain).
	if err := tx.Model(&models.Webhook{}).
		Where("id = ?", duplicateID).
		Updates(map[string]any{
			"provisioning_mode": models.WebhookProvisioningModeOps,
			"updated_at":        now,
		}).Error; err != nil {
		return fmt.Errorf("marking duplicate as ops-mode: %w", err)
	}

	return nil
}

func (j *WebhookDedupeJob) log(format string, v ...any) {
	log.Printf("[WebhookDedupeJob] "+format, v...)
}
