package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WebhookStatePending        = "pending"
	WebhookStateProvisioning   = "provisioning"
	WebhookStateReady          = "ready"
	WebhookStateFailed         = "failed"
	WebhookStateDeletingPending = "deleting_pending"

	WebhookProvisioningModeLegacy = "legacy"
	WebhookProvisioningModeOps    = "ops"
)

type Webhook struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	State             string
	Secret            []byte
	Configuration     datatypes.JSONType[any]
	Metadata          datatypes.JSONType[any]
	AppInstallationID *uuid.UUID
	RetryCount        int `gorm:"default:0"`
	MaxRetries        int `gorm:"default:3"`
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`

	// Phase 0: registration identity and error tracking (all nullable; unused by legacy path).
	ScopeKey           *string
	ConfigHash         *string
	ProviderWebhookID  *string
	ProviderEtag       *string
	SecretVersion      int `gorm:"default:1"`
	LastProvisionedAt  *time.Time
	LastErrorCode      *string
	LastErrorMessage   *string
	LastErrorAt        *time.Time

	// Phase 2: which provisioning path owns this registration.
	// 'legacy' = legacy webhooks.state polling; 'ops' = webhook_operations path.
	ProvisioningMode string `gorm:"default:'legacy'"`
}

type WebhookResource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (w *Webhook) Ready(tx *gorm.DB) error {
	return tx.Model(w).
		Update("state", WebhookStateReady).
		Update("updated_at", time.Now()).
		Error
}

func (w *Webhook) ReadyWithMetadata(tx *gorm.DB, metadata any) error {
	return tx.Model(w).
		Update("state", WebhookStateReady).
		Update("metadata", datatypes.NewJSONType(metadata)).
		Update("updated_at", time.Now()).
		Error
}

func (w *Webhook) MarkProvisioning(tx *gorm.DB) error {
	return tx.Model(w).
		Update("state", WebhookStateProvisioning).
		Update("updated_at", time.Now()).
		Error
}

func (w *Webhook) IncrementRetry(tx *gorm.DB) error {
	w.RetryCount++
	return tx.Model(w).
		Update("retry_count", w.RetryCount).
		Update("updated_at", time.Now()).
		Error
}

func (w *Webhook) MarkFailed(tx *gorm.DB) error {
	return tx.Model(w).
		Update("state", WebhookStateFailed).
		Update("updated_at", time.Now()).
		Error
}

func (w *Webhook) HasExceededRetries() bool {
	return w.RetryCount >= w.MaxRetries
}

// FindWebhookByScope returns the active webhook for a given (app_installation_id, scope_key)
// pair. Returns gorm.ErrRecordNotFound when no match exists, which callers interpret as
// "no registration yet."
func FindWebhookByScope(tx *gorm.DB, appInstallationID uuid.UUID, scopeKey string) (*Webhook, error) {
	var webhook Webhook
	err := tx.
		Where("app_installation_id = ? AND scope_key = ?", appInstallationID, scopeKey).
		First(&webhook).
		Error
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

func FindWebhook(id uuid.UUID) (*Webhook, error) {
	var webhook Webhook
	err := database.Conn().
		First(&webhook, id).
		Error

	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func FindWebhookInTransaction(tx *gorm.DB, id uuid.UUID) (*Webhook, error) {
	var webhook Webhook
	err := tx.
		First(&webhook, id).
		Error

	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func FindWebhookNodes(webhookID uuid.UUID) ([]CanvasNode, error) {
	return FindWebhookNodesInTransaction(database.Conn(), webhookID)
}

func FindActiveWebhookNodes(webhookID uuid.UUID) ([]CanvasNode, error) {
	return FindActiveWebhookNodesInTransaction(database.Conn(), webhookID)
}

func FindWebhookNodesInTransaction(tx *gorm.DB, webhookID uuid.UUID) ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := tx.
		Where("webhook_id = ?", webhookID).
		Where("deleted_at IS NULL").
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func FindActiveWebhookNodesInTransaction(tx *gorm.DB, webhookID uuid.UUID) ([]CanvasNode, error) {
	var nodes []CanvasNode
	query := tx.
		Table("workflow_nodes").
		Select("workflow_nodes.*").
		Where("workflow_nodes.webhook_id = ?", webhookID).
		Where("workflow_nodes.deleted_at IS NULL")

	err := withActiveCanvas(query, "workflow_nodes.workflow_id").
		Find(&nodes).
		Error

	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func ListPendingWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	err := database.Conn().
		Where("state = ? AND provisioning_mode = ?", WebhookStatePending, WebhookProvisioningModeLegacy).
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}

	return webhooks, nil
}

// ListOrphanedOpsWebhooks returns ops-mode registrations that have no active
// WebhookSubscriptionBindings and are not already in a terminal / deleting state.
// The reconciler uses this to detect and schedule deletion of abandoned registrations.
func ListOrphanedOpsWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	err := database.Conn().
		Where(`provisioning_mode = ?
			AND state NOT IN (?, ?, ?)
			AND NOT EXISTS (
				SELECT 1 FROM webhook_subscription_bindings
				WHERE webhook_subscription_bindings.webhook_id = webhooks.id
				  AND webhook_subscription_bindings.active = true
				  AND webhook_subscription_bindings.deleted_at IS NULL
			)`,
			WebhookProvisioningModeOps,
			WebhookStateDeletingPending, WebhookStateFailed, "deleted",
		).
		Find(&webhooks).
		Error
	return webhooks, err
}

// ListOpsWebhooksWithoutPendingOp returns ops-mode registrations whose state
// suggests they need a new operation but none is currently queued or running.
// Used by the reconciler to recover registrations that lost their operation
// (e.g. process crash between registration creation and operation insert).
func ListOpsWebhooksWithoutPendingOp() ([]Webhook, error) {
	var webhooks []Webhook
	err := database.Conn().
		Where(`provisioning_mode = ?
			AND state IN (?, ?)
			AND NOT EXISTS (
				SELECT 1 FROM webhook_operations
				WHERE webhook_operations.webhook_id = webhooks.id
				  AND webhook_operations.state IN (?, ?)
			)`,
			WebhookProvisioningModeOps,
			WebhookStatePending, WebhookStateDeletingPending,
			WebhookOperationStateQueued, WebhookOperationStateRunning,
		).
		Find(&webhooks).
		Error
	return webhooks, err
}

// ListDeletedWebhooks returns soft-deleted legacy-mode webhooks that still need
// provider-side cleanup. Ops-mode webhooks are excluded because the ops provisioner
// calls handler.Cleanup before soft-deleting them; processing them here would result
// in a redundant provider call.
func ListDeletedWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	err := database.Conn().Unscoped().
		Where("deleted_at IS NOT NULL AND provisioning_mode = ?", WebhookProvisioningModeLegacy).
		Find(&webhooks).
		Error
	return webhooks, err
}

func LockWebhook(tx *gorm.DB, ID uuid.UUID) (*Webhook, error) {
	var webhook Webhook

	err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", ID).
		Where("state = ?", WebhookStatePending).
		First(&webhook).
		Error

	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

// LockDeletedWebhook acquires a row-level lock on a soft-deleted webhook
// regardless of its state. Used by WebhookCleanupWorker to clean up
// webhooks that were in any state (ready, failed, etc.) when deleted.
func LockDeletedWebhook(tx *gorm.DB, ID uuid.UUID) (*Webhook, error) {
	var webhook Webhook

	err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", ID).
		Where("deleted_at IS NOT NULL").
		First(&webhook).
		Error

	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

// ResetStuckProvisioningWebhooks resets webhooks that have been stuck in
// "provisioning" state back to "pending". This handles the edge case where
// the process crashes during the external API call (Phase 2).
func ResetStuckProvisioningWebhooks() (int64, error) {
	result := database.Conn().
		Model(&Webhook{}).
		Where("state = ?", WebhookStateProvisioning).
		Updates(map[string]any{
			"state":      WebhookStatePending,
			"updated_at": time.Now(),
		})

	return result.RowsAffected, result.Error
}
