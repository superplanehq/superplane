package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WebhookSubscriptionBinding struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID    uuid.UUID
	AppInstallationID uuid.UUID
	WorkflowID        uuid.UUID
	NodeID            string
	WebhookID         *uuid.UUID
	ScopeKey          string
	RequestedConfig   datatypes.JSONType[any]
	RequestedHash     string
	Active            bool `gorm:"default:true"`
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// BindingGroup is a distinct (app_installation_id, scope_key) pair that the
// shadow reconciler and future reconciler use as their unit of work.
type BindingGroup struct {
	AppInstallationID uuid.UUID
	ScopeKey          string
}

// ListActiveBindingGroups returns the distinct (app_installation_id, scope_key)
// pairs that have at least one active binding. Used by the reconciler to enumerate
// reconciliation units without loading all binding rows at once.
func ListActiveBindingGroups() ([]BindingGroup, error) {
	var groups []BindingGroup
	err := database.Conn().
		Model(&WebhookSubscriptionBinding{}).
		Select("app_installation_id, scope_key").
		Where("active = true").
		Group("app_installation_id, scope_key").
		Scan(&groups).
		Error
	return groups, err
}

// ListActiveBindingsForGroup returns all active bindings for a reconciliation
// group, ordered by id ascending for deterministic merge order.
func ListActiveBindingsForGroup(appInstallationID uuid.UUID, scopeKey string) ([]WebhookSubscriptionBinding, error) {
	var bindings []WebhookSubscriptionBinding
	err := database.Conn().
		Where("app_installation_id = ? AND scope_key = ? AND active = true", appInstallationID, scopeKey).
		Order("id ASC").
		Find(&bindings).
		Error
	return bindings, err
}
