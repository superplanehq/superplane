package models

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlanFeatureDefinition struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Label string `json:"label"`
	Gated bool   `json:"gated"`
}

// FeatureOrganization lists saved access sources without resolving billing state.
type FeatureOrganization struct {
	ID           uuid.UUID                `json:"id"`
	Name         string                   `json:"name"`
	Slug         string                   `json:"slug"`
	EnabledGrant bool                     `json:"enabled_grant"`
	Plan         *FeatureOrganizationPlan `json:"plan" gorm:"embedded;embeddedPrefix:plan_"`
}

type FeatureOrganizationPlan struct {
	ID   uuid.UUID `json:"id"`
	Key  string    `json:"key"`
	Name string    `json:"name"`
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

// DeletePlanFeatureDefinition removes a feature key from every billing plan,
// deletes every organization grant for the key, and deletes the definition.
// The steps run in one transaction, so a failure leaves all records unchanged.
func DeletePlanFeatureDefinition(tx *gorm.DB, key string) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		var feature PlanFeatureDefinition
		if err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key = ?", key).First(&feature).Error; err != nil {
			return err
		}
		if err := inner.Exec(`
			UPDATE billing_plans
			SET features = COALESCE((
				SELECT jsonb_agg(element ORDER BY position)
				FROM jsonb_array_elements_text(billing_plans.features) WITH ORDINALITY AS entries(element, position)
				WHERE element <> ?
			), '[]'::jsonb),
			updated_at = now()
			WHERE jsonb_exists(features, ?)`, key, key).Error; err != nil {
			return err
		}
		if err := inner.Where("feature_key = ?", key).Delete(&OrganizationFeatureGrant{}).Error; err != nil {
			return err
		}
		return inner.Delete(&feature).Error
	})
}

func ListFeatureOrganizations(tx *gorm.DB, featureKey, search string, limit, offset int) ([]FeatureOrganization, int64, error) {
	var feature PlanFeatureDefinition
	if err := tx.Where("key = ?", featureKey).First(&feature).Error; err != nil {
		return nil, 0, err
	}
	query := organizationsMatchingSearch(tx, search).
		Joins("LEFT JOIN organization_feature_grants AS grants ON grants.organization_id = organizations.id AND grants.feature_key = ? AND grants.enabled = true", featureKey).
		Joins("LEFT JOIN organization_billing_plans AS assigned ON assigned.organization_id = organizations.id").
		Joins("LEFT JOIN billing_plans AS plans ON plans.id = assigned.plan_id AND jsonb_exists(plans.features, ?)", featureKey).
		Where("grants.organization_id IS NOT NULL OR plans.id IS NOT NULL")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []FeatureOrganization{}
	err := query.Select("organizations.id, organizations.name, organizations.slug, (grants.organization_id IS NOT NULL) AS enabled_grant, plans.id AS plan_id, plans.key AS plan_key, plans.name AS plan_name").
		Order("organizations.name, organizations.id").Limit(limit).Offset(offset).Scan(&items).Error
	return items, total, err
}

// lockPlanFeatureReferences validates keys and holds their shared row locks until
// the reference write commits. Deletion takes an exclusive lock on the same row.
// Callers must run this before locking plans to keep the lock order consistent.
func lockPlanFeatureReferences(tx *gorm.DB, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	keys = slices.Clone(keys)
	slices.Sort(keys)
	keys = slices.Compact(keys)
	var features []PlanFeatureDefinition
	if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("key IN ?", keys).Order("key").Find(&features).Error; err != nil {
		return err
	}
	if len(features) != len(keys) {
		return fmt.Errorf("%w: unknown feature", ErrCatalogPlan)
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

func organizationsMatchingSearch(tx *gorm.DB, search string) *gorm.DB {
	query := tx.Model(&coremodels.Organization{}).Where("organizations.deleted_at IS NULL")
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}
	pattern := "%" + search + "%"
	return query.Where("organizations.name ILIKE ? OR organizations.id::text ILIKE ?", pattern, pattern)
}
