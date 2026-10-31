package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IntegrationModelCatalog records provider discovery, never execution authority.
// A missing row means unknown availability; an empty Models slice means the
// provider successfully returned no models at ObservedAt.
type IntegrationModelCatalog struct {
	IntegrationID uuid.UUID `gorm:"primaryKey"`
	Models        datatypes.JSONSlice[SelectableLLMNamedID]
	ObservedAt    time.Time
}

func (IntegrationModelCatalog) TableName() string { return "integration_model_catalogs" }

func (c *IntegrationModelCatalog) ModelNames() map[string]string {
	names := map[string]string{}
	if c == nil {
		return names
	}
	for _, model := range c.Models {
		names[model.ID] = model.Name
	}
	return names
}

func FindIntegrationModelCatalog(tx *gorm.DB, integrationID uuid.UUID) (*IntegrationModelCatalog, error) {
	var catalog IntegrationModelCatalog
	err := tx.First(&catalog, "integration_id = ?", integrationID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &catalog, nil
}

// FindProviderModelCatalog returns discovery for the currently connected key.
func FindProviderModelCatalog(tx *gorm.DB, orgID uuid.UUID, provider string) (*IntegrationModelCatalog, error) {
	integration, err := FindReadyBYOKIntegration(tx, orgID, provider)
	if err != nil || integration == nil {
		return nil, err
	}
	return FindIntegrationModelCatalog(tx, integration.ID)
}

func StoreIntegrationModelCatalog(tx *gorm.DB, integrationID uuid.UUID, models []SelectableLLMNamedID) error {
	if models == nil {
		models = []SelectableLLMNamedID{}
	}
	catalog := IntegrationModelCatalog{IntegrationID: integrationID, Models: models, ObservedAt: time.Now()}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "integration_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"models", "observed_at"}),
	}).Create(&catalog).Error
}
