package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const LLMCatalogIntegrationBatchSize = 100

// ListClaudeCatalogIntegrations returns ready connections for active organizations.
func ListClaudeCatalogIntegrations(tx *gorm.DB, after uuid.UUID) ([]Integration, error) {
	var integrations []Integration
	err := tx.Model(&Integration{}).
		Joins("JOIN organizations ON organizations.id = app_installations.organization_id").
		Where("app_installations.app_name = ? AND app_installations.state = ?", "claude", IntegrationStateReady).
		Where("organizations.deleted_at IS NULL AND organizations.suspended_at IS NULL").
		Where("app_installations.id > ?", after).
		Order("app_installations.id ASC").
		Limit(LLMCatalogIntegrationBatchSize).
		Find(&integrations).Error
	return integrations, err
}
