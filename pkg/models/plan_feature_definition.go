package models

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlanFeatureDefinition struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Label string `json:"label"`
	Gated bool   `json:"gated"`
}

func ListPlanFeatureDefinitions(tx *gorm.DB) ([]PlanFeatureDefinition, error) {
	features := []PlanFeatureDefinition{}
	err := tx.Order("key").Find(&features).Error
	return features, err
}

func CreatePlanFeatureDefinition(tx *gorm.DB, feature *PlanFeatureDefinition) error {
	if err := validatePlanFeatureDefinition(feature); err != nil {
		return err
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(feature)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: feature key already exists", ErrCatalogPlan)
	}
	return nil
}

func UpdatePlanFeatureDefinition(tx *gorm.DB, feature *PlanFeatureDefinition) error {
	if err := validatePlanFeatureDefinition(feature); err != nil {
		return err
	}
	result := tx.Model(&PlanFeatureDefinition{}).Where("key = ?", feature.Key).Updates(map[string]any{"label": feature.Label, "gated": feature.Gated})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func validatePlanFeatureDefinition(feature *PlanFeatureDefinition) error {
	feature.Label = strings.TrimSpace(feature.Label)
	if !catalogKeyPattern.MatchString(feature.Key) || feature.Label == "" {
		return fmt.Errorf("%w: a valid feature key and label are required", ErrCatalogPlan)
	}
	return nil
}
