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
	LineID         uuid.UUID `gorm:"primaryKey"`
	StageKey       string    `gorm:"primaryKey"`
	ManifestID     string    `gorm:"primaryKey"`
	CanvasID       uuid.UUID
	Binding        datatypes.JSONMap
}

func (FactoryConfigurationAutomation) TableName() string {
	return "factory_configuration_automations"
}

func (a *FactoryConfigurationAutomation) Save(tx *gorm.DB) error {
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "factory_id"}, {Name: "line_id"}, {Name: "stage_key"}, {Name: "manifest_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"canvas_id", "binding"}),
	}).Create(a).Error; err != nil {
		return err
	}
	// A canvas has one runtime binding even when several lines own identities.
	return tx.Model(&FactoryConfigurationAutomation{}).
		Where("organization_id = ? AND factory_id = ? AND canvas_id = ? AND binding IS DISTINCT FROM ?", a.OrganizationID, a.FactoryID, a.CanvasID, a.Binding).
		Update("binding", a.Binding).Error
}

func ListFactoryConfigurationAutomations(tx *gorm.DB, organizationID, factoryID, lineID uuid.UUID) ([]FactoryConfigurationAutomation, error) {
	var records []FactoryConfigurationAutomation
	query := tx.Where("organization_id = ? AND factory_id = ?", organizationID, factoryID)
	if lineID != uuid.Nil {
		query = query.Where("line_id = ?", lineID)
	}
	err := query.Order("line_id, stage_key, manifest_id").Find(&records).Error
	return records, err
}

func FindFactoryConfigurationAutomationByCanvas(tx *gorm.DB, canvasID uuid.UUID) (*FactoryConfigurationAutomation, error) {
	var record FactoryConfigurationAutomation
	err := tx.Where("canvas_id = ?", canvasID).Order("line_id, stage_key, manifest_id").First(&record).Error
	return &record, err
}
