package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

// WebhookShadowReconciler groups active WebhookSubscriptionBindings by
// (app_installation_id, scope_key), computes the deterministic desired merged
// config for each group, and logs any drift against the current registration.
//
// Phase 1: audit-only — no operations are enqueued and no state is mutated.
type WebhookShadowReconciler struct {
	registry *registry.Registry
	interval time.Duration
}

func NewWebhookShadowReconciler(registry *registry.Registry) *WebhookShadowReconciler {
	return &WebhookShadowReconciler{
		registry: registry,
		interval: time.Minute,
	}
}

func (r *WebhookShadowReconciler) Start(ctx context.Context) {
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

func (r *WebhookShadowReconciler) reconcileAll() {
	groups, err := models.ListActiveBindingGroups()
	if err != nil {
		r.log("Error listing binding groups: %v", err)
		return
	}

	if len(groups) == 0 {
		return
	}

	r.log("Reconciling %d binding group(s)", len(groups))

	for _, group := range groups {
		if err := r.reconcileGroup(group); err != nil {
			r.log("Error reconciling group app_installation=%s scope=%q: %v",
				group.AppInstallationID, group.ScopeKey, err)
		}
	}
}

func (r *WebhookShadowReconciler) reconcileGroup(group models.BindingGroup) error {
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

	// Find the current registration for this (app_installation_id, scope_key).
	// In Phase 1 most registrations won't have scope_key set yet, so this is
	// expected to return ErrRecordNotFound frequently.
	current, err := models.FindWebhookByScope(db, group.AppInstallationID, group.ScopeKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log("No scoped registration for app_installation=%s scope=%q — %d binding(s), desired ready when reconciler creates one",
				group.AppInstallationID, group.ScopeKey, len(bindings))
			return nil
		}
		return fmt.Errorf("finding registration: %w", err)
	}

	matches, err := handler.CompareConfig(current.Configuration.Data(), desired)
	if err != nil {
		return fmt.Errorf("comparing configs: %w", err)
	}

	if !matches {
		r.log("Drift detected: app_installation=%s scope=%q webhook=%s — current config does not match desired from %d binding(s)",
			group.AppInstallationID, group.ScopeKey, current.ID, len(bindings))
	}

	return nil
}

func (r *WebhookShadowReconciler) log(format string, v ...any) {
	log.Printf("[WebhookShadowReconciler] "+format, v...)
}
