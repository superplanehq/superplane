package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

const (
	WebhookAuthenticationBearer      = "bearer"
	WebhookAuthenticationSignature   = "signature"
	WebhookAuthenticationHeaderToken = "header_token"
)

type Webhook struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	State             string
	Secret            []byte
	Configuration     datatypes.JSONType[any]
	Metadata          datatypes.JSONType[any]
	AppInstallationID *uuid.UUID
	OrganizationID    *uuid.UUID
	FactoryID         *uuid.UUID
	Name              *string
	RetryCount        int `gorm:"default:0"`
	MaxRetries        int `gorm:"default:3"`
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// WebhookAuthenticationConfig is the configuration of a webhook that a user
// manages in workspace settings.
type WebhookAuthenticationConfig struct {
	Authentication string `json:"authentication" mapstructure:"authentication"`
	HeaderName     string `json:"headerName,omitempty" mapstructure:"headerName"`
}

var ErrWebhookNameAlreadyExists = errors.New("webhook name already exists")

const webhookFactoryNameConstraint = "idx_webhooks_factory_name"

// NewFactoryWebhook builds a webhook that a user manages in workspace settings.
// The secret must already be encrypted with the webhook ID as associated data.
func NewFactoryWebhook(id, organizationID, factoryID uuid.UUID, name string, config WebhookAuthenticationConfig, encryptedSecret []byte) Webhook {
	now := time.Now()
	return Webhook{
		ID:             id,
		State:          WebhookStateReady,
		Secret:         encryptedSecret,
		Configuration:  datatypes.NewJSONType[any](config),
		Metadata:       datatypes.NewJSONType[any](map[string]any{}),
		OrganizationID: &organizationID,
		FactoryID:      &factoryID,
		Name:           &name,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
}

// IsManaged reports whether a user manages this webhook in workspace settings.
// A managed webhook outlives the automations that use it.
func (w *Webhook) IsManaged() bool {
	return w.FactoryID != nil
}

func (w *Webhook) DisplayName() string {
	if w.Name == nil {
		return ""
	}
	return *w.Name
}

func (w *Webhook) AuthenticationConfig() WebhookAuthenticationConfig {
	var config WebhookAuthenticationConfig
	raw, err := json.Marshal(w.Configuration.Data())
	if err != nil {
		return config
	}
	_ = json.Unmarshal(raw, &config)
	return config
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
	return w.Fail(tx, "")
}

// Fail marks the webhook failed and stores the setup error for the factory UI.
// The error merges into the existing metadata, so registration IDs from a
// previous ready state stay available for cleanup. An empty reason leaves
// metadata unchanged.
func (w *Webhook) Fail(tx *gorm.DB, reason string) error {
	updates := map[string]any{
		"state":      WebhookStateFailed,
		"updated_at": time.Now(),
	}
	reason = strings.TrimSpace(reason)
	if reason != "" {
		if len(reason) > 500 {
			reason = reason[:500]
		}
		setupError, err := json.Marshal(map[string]any{"setupError": reason})
		if err != nil {
			return err
		}
		updates["metadata"] = gorm.Expr(
			"CASE WHEN jsonb_typeof(metadata) = 'object' THEN metadata ELSE '{}'::jsonb END || ?::jsonb",
			string(setupError),
		)
	}
	return tx.Model(w).Updates(updates).Error
}

func (w *Webhook) HasExceededRetries() bool {
	return w.RetryCount >= w.MaxRetries
}

func (w *Webhook) HardDelete(tx *gorm.DB) error {
	return tx.Unscoped().Delete(w).Error
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
	if webhook.IsManaged() {
		return nil
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

func (w *Webhook) Create(tx *gorm.DB) error {
	return mapWebhookNameError(tx.Create(w).Error)
}

func (w *Webhook) UpdateSettings(tx *gorm.DB, name string, config WebhookAuthenticationConfig) error {
	w.Name = &name
	w.Configuration = datatypes.NewJSONType[any](config)
	return mapWebhookNameError(tx.Model(w).Updates(map[string]any{
		"name":          name,
		"configuration": w.Configuration,
		"updated_at":    time.Now(),
	}).Error)
}

func (w *Webhook) ReplaceSecret(tx *gorm.DB, encrypted []byte) error {
	w.Secret = encrypted
	return tx.Model(w).Updates(map[string]any{
		"secret":     encrypted,
		"updated_at": time.Now(),
	}).Error
}

// DeleteUnlessListening soft-deletes a workspace webhook when no live node listens on it.
// The row lock conflicts with the share lock that a binding node takes, so a
// concurrent publish cannot attach a node to a deleted webhook.
func (w *Webhook) DeleteUnlessListening(tx *gorm.DB) (bool, error) {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", w.ID).First(w).Error; err != nil {
		return false, err
	}
	counts, err := CountActiveWebhookNodes(tx, []uuid.UUID{w.ID})
	if err != nil {
		return false, err
	}
	if counts[w.ID] > 0 {
		return false, nil
	}
	return true, tx.Delete(w).Error
}

func ListFactoryWebhooks(tx *gorm.DB, organizationID, factoryID uuid.UUID) ([]Webhook, error) {
	var webhooks []Webhook
	err := tx.
		Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).
		Order("name ASC").
		Find(&webhooks).
		Error
	return webhooks, err
}

func FindFactoryWebhook(tx *gorm.DB, organizationID, factoryID, id uuid.UUID) (*Webhook, error) {
	var webhook Webhook
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND id = ?", organizationID, factoryID, id).
		First(&webhook).
		Error
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

// CountActiveWebhookNodes returns, per webhook, how many live nodes listen on it.
func CountActiveWebhookNodes(tx *gorm.DB, webhookIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(webhookIDs))
	if len(webhookIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		WebhookID uuid.UUID
		Count     int
	}
	query := tx.
		Table("workflow_nodes").
		Select("workflow_nodes.webhook_id AS webhook_id, COUNT(*) AS count").
		Where("workflow_nodes.webhook_id IN ?", webhookIDs).
		Where("workflow_nodes.deleted_at IS NULL").
		Group("workflow_nodes.webhook_id")
	if err := withActiveCanvas(query, "workflow_nodes.workflow_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.WebhookID] = row.Count
	}
	return counts, nil
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
		Where(
			"(trim(both '{}' from webhooks.metadata->>'repositoryUUID') = ? OR EXISTS ("+
				"SELECT 1 FROM jsonb_array_elements(COALESCE(webhooks.metadata->'hooks', '[]'::jsonb)) AS hook "+
				"WHERE trim(both '{}' from hook->>'repositoryUUID') = ?))",
			repositoryUUID, repositoryUUID,
		).
		Find(&webhooks).Error
	return webhooks, err
}

func ListPendingWebhooks(tx *gorm.DB, integrationNames []string) ([]Webhook, error) {
	var webhooks []Webhook
	err := withRegisteredWebhookIntegration(tx, integrationNames).
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

func LockWebhook(tx *gorm.DB, ID uuid.UUID, integrationNames []string) (*Webhook, error) {
	var webhook Webhook

	err := withRegisteredWebhookIntegration(tx, integrationNames).
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
func ResetStuckProvisioningWebhooks(tx *gorm.DB, integrationNames []string) (int64, error) {
	result := withRegisteredWebhookIntegration(tx, integrationNames).
		Model(&Webhook{}).
		Where("state = ?", WebhookStateProvisioning).
		Updates(map[string]any{
			"state":      WebhookStatePending,
			"updated_at": time.Now(),
		})

	return result.RowsAffected, result.Error
}

func withRegisteredWebhookIntegration(tx *gorm.DB, integrationNames []string) *gorm.DB {
	return tx.Where("app_installation_id IS NULL OR app_installation_id IN (?)", registeredIntegrationIDs(tx, integrationNames))
}

func mapWebhookNameError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == webhookFactoryNameConstraint {
		return ErrWebhookNameAlreadyExists
	}
	return err
}
