package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FactoryConfigurationAutomation keeps manifest identity and explicit binding
// intent independent of a canvas name or a generated template alias.
type FactoryConfigurationAutomation struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID `gorm:"primaryKey"`
	LineID         uuid.UUID
	StageKey       string
	ManifestID     string `gorm:"primaryKey"`
	CanvasID       uuid.UUID
	Binding        datatypes.JSONMap
}

func (FactoryConfigurationAutomation) TableName() string {
	return "factory_configuration_automations"
}

func (a *FactoryConfigurationAutomation) Save(tx *gorm.DB) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "factory_id"}, {Name: "manifest_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"line_id", "stage_key", "canvas_id", "binding"}),
	}).Create(a).Error
}

func ListFactoryConfigurationAutomations(tx *gorm.DB, organizationID, factoryID uuid.UUID) ([]FactoryConfigurationAutomation, error) {
	var records []FactoryConfigurationAutomation
	query := tx.Where("organization_id = ? AND factory_id = ?", organizationID, factoryID)
	err := query.Order("line_id, stage_key, manifest_id").Find(&records).Error
	return records, err
}

func FindFactoryConfigurationAutomationByCanvas(tx *gorm.DB, canvasID uuid.UUID) (*FactoryConfigurationAutomation, error) {
	var record FactoryConfigurationAutomation
	err := tx.Where("canvas_id = ?", canvasID).Order("line_id, stage_key, manifest_id").First(&record).Error
	return &record, err
}
