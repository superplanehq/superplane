package models

import (
	"errors"
	"fmt"
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

// UpdateConfiguration stores a new configuration and leaves the state unchanged.
func (w *Webhook) UpdateConfiguration(tx *gorm.DB, configuration any) error {
	if w == nil || w.ID == uuid.Nil {
		return fmt.Errorf("missing webhook id")
	}

	w.Configuration = datatypes.NewJSONType[any](configuration)
	return tx.Model(w).Updates(map[string]any{
		"configuration": w.Configuration,
		"updated_at":    time.Now(),
	}).Error
}

// Reprovision marks a ready or failed webhook pending so Setup runs again.
// A webhook that is already pending or provisioning is left unchanged, so a
// second sync cannot start another remote webhook while the first is in progress.
// The bool is true when this call changed the row.
func (w *Webhook) Reprovision(tx *gorm.DB, configuration any) (bool, error) {
	if w == nil || w.ID == uuid.Nil {
		return false, fmt.Errorf("missing webhook id")
	}

	result := tx.Model(&Webhook{}).
		Where("id = ?", w.ID).
		Where("state IN ?", []string{WebhookStateReady, WebhookStateFailed}).
		Updates(map[string]any{
			"state":         WebhookStatePending,
			"retry_count":   0,
			"configuration": datatypes.NewJSONType[any](configuration),
			"updated_at":    time.Now(),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
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

// SoftDeleteWebhookIfUnreferenced soft-deletes webhookID when no live canvas
// node still references it. Live means the node, canvas, and organization are
// not deleted.
func SoftDeleteWebhookIfUnreferenced(tx *gorm.DB, webhookID uuid.UUID) error {
	webhook, err := FindWebhookInTransaction(tx, webhookID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	nodes, err := FindActiveWebhookNodesInTransaction(tx, webhookID)
	if err != nil {
		return err
	}
	if len(nodes) > 0 {
		return nil
	}

	return tx.Delete(webhook).Error
}

func ListBitbucketForgeWebhooks(tx *gorm.DB, installationID, repositoryUUID string) ([]Webhook, error) {
	var webhooks []Webhook
	err := tx.Model(&Webhook{}).
		Select("webhooks.*").
		Joins("JOIN app_installations ON app_installations.id = webhooks.app_installation_id").
		Where("app_installations.deleted_at IS NULL AND app_installations.app_name = ?", "bitbucket").
		Where("app_installations.metadata->>'forgeInstallationId' = ?", installationID).
		Where("app_installations.metadata->>'authType' = ?", "forgeApp").
		Where("webhooks.state = ?", WebhookStateReady).
		Where("trim(both '{}' from webhooks.metadata->>'repositoryUUID') = ?", repositoryUUID).
		Find(&webhooks).Error
	return webhooks, err
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
