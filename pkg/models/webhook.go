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
	WebhookStatePending      = "pending"
	WebhookStateProvisioning = "provisioning"
	WebhookStateReady        = "ready"
	WebhookStateFailed       = "failed"
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

func ListPendingWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	err := database.Conn().
		Where("state = ?", WebhookStatePending).
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}

	return webhooks, nil
}

func ListDeletedWebhooks() ([]Webhook, error) {
	var webhooks []Webhook
	err := database.Conn().Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&webhooks).
		Error

	if err != nil {
		return nil, err
	}

	return webhooks, nil
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
