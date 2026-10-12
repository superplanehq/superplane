package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrganizationHostedModelAllowlist narrows the installation list for one provider.
// A missing row inherits the installation list. An empty saved list allows no models.
type OrganizationHostedModelAllowlist struct {
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	Provider       string    `gorm:"primaryKey"`
	AllowedModels  datatypes.JSONSlice[string]
	UpdatedAt      time.Time
}

func (OrganizationHostedModelAllowlist) TableName() string {
	return "organization_hosted_model_allowlists"
}

func FindOrganizationHostedModelAllowlist(tx *gorm.DB, orgID uuid.UUID, provider string) (*OrganizationHostedModelAllowlist, error) {
	normalized, err := NormalizeHostedLLMProvider(provider)
	if err != nil {
		return nil, err
	}
	var row OrganizationHostedModelAllowlist
	err = tx.Where("organization_id = ? AND provider = ?", orgID, normalized).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func UpsertOrganizationHostedModelAllowlist(tx *gorm.DB, orgID uuid.UUID, provider string, models datatypes.JSONSlice[string]) (*OrganizationHostedModelAllowlist, error) {
	normalized, err := NormalizeHostedLLMProvider(provider)
	if err != nil {
		return nil, err
	}
	normalizedModels, err := normalizeAllowedModels(models)
	if err != nil {
		return nil, err
	}
	row := OrganizationHostedModelAllowlist{
		OrganizationID: orgID, Provider: normalized, AllowedModels: normalizedModels, UpdatedAt: time.Now(),
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"allowed_models", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
