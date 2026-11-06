package models

import "gorm.io/gorm"

// ListClaudeCatalogIntegrations returns ready connections for active organizations.
func ListClaudeCatalogIntegrations(tx *gorm.DB) ([]Integration, error) {
	var integrations []Integration
	err := tx.Model(&Integration{}).
		Joins("JOIN organizations ON organizations.id = app_installations.organization_id").
		Where("app_installations.app_name = ? AND app_installations.state = ?", "claude", IntegrationStateReady).
		Where("organizations.deleted_at IS NULL AND organizations.suspended_at IS NULL").
		Order("app_installations.created_at ASC, app_installations.id ASC").
		Find(&integrations).Error
	return integrations, err
}
