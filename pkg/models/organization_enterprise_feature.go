package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	EnterpriseFeatureSourceAdmin         = "admin"
	EnterpriseFeatureSourceAutoSingleOrg = "auto_single_org"

	EnterpriseFeatureRemovedByActivation   = "activation"
	EnterpriseFeatureRemovedByDeactivation = "deactivation"
)

// OrganizationEnterpriseFeature enables one license feature for one organization.
// A row exists only while the grant is enabled.
type OrganizationEnterpriseFeature struct {
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	FeatureKey     string    `gorm:"primaryKey"`
	GrantedBy      *uuid.UUID
	Source         string
	CreatedAt      time.Time
}

func (OrganizationEnterpriseFeature) TableName() string {
	return "organization_enterprise_features"
}

func HasOrganizationEnterpriseFeature(tx *gorm.DB, orgID uuid.UUID, key string) (bool, error) {
	if orgID == uuid.Nil || key == "" {
		return false, nil
	}

	var row OrganizationEnterpriseFeature
	err := tx.
		Select("organization_id").
		Where("organization_id = ? AND feature_key = ?", orgID, key).
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func ListAllOrganizationEnterpriseFeatures(tx *gorm.DB) ([]OrganizationEnterpriseFeature, error) {
	var rows []OrganizationEnterpriseFeature
	err := tx.Order("organization_id ASC, feature_key ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func ListOrganizationEnterpriseFeatures(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationEnterpriseFeature, error) {
	var rows []OrganizationEnterpriseFeature
	err := tx.Where("organization_id = ?", orgID).Order("feature_key ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// ListOrganizationsWithEnterpriseFeature returns active organizations that
// have a grant for key. Deleted organizations are excluded.
func ListOrganizationsWithEnterpriseFeature(tx *gorm.DB, key string, limit, offset int) ([]uuid.UUID, int64, error) {
	base := tx.Model(&OrganizationEnterpriseFeature{}).
		Joins("JOIN organizations ON organizations.id = organization_enterprise_features.organization_id AND organizations.deleted_at IS NULL").
		Where("organization_enterprise_features.feature_key = ?", key)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base.Session(&gorm.Session{}).Order("organization_enterprise_features.created_at ASC, organization_enterprise_features.organization_id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var ids []uuid.UUID
	err := query.Pluck("organization_enterprise_features.organization_id", &ids).Error
	if err != nil {
		return nil, 0, err
	}

	return ids, total, nil
}

func CountOrganizationsByEnterpriseFeature(tx *gorm.DB) (map[string]int64, error) {
	var rows []struct {
		FeatureKey string
		Count      int64
	}
	err := tx.Model(&OrganizationEnterpriseFeature{}).
		Select("organization_enterprise_features.feature_key, COUNT(*) AS count").
		Joins("JOIN organizations ON organizations.id = organization_enterprise_features.organization_id AND organizations.deleted_at IS NULL").
		Group("organization_enterprise_features.feature_key").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.FeatureKey] = row.Count
	}

	return counts, nil
}

// SetOrganizationEnterpriseFeatures replaces the grants of one organization.
// Rows that stay keep their source, granted_by, and created_at. New rows use
// the admin source. The caller validates the keys. The organization row stays
// locked until the transaction ends, so two replacements cannot merge.
func SetOrganizationEnterpriseFeatures(tx *gorm.DB, orgID uuid.UUID, keys []string, grantedBy uuid.UUID) (added, removed []OrganizationEnterpriseFeature, err error) {
	err = tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT id FROM organizations WHERE id = ? FOR UPDATE", orgID).Error; err != nil {
			return err
		}

		var replaceErr error
		added, removed, replaceErr = replaceOrganizationEnterpriseFeatures(tx, orgID, keys, grantedBy)
		return replaceErr
	})
	if err != nil {
		return nil, nil, err
	}

	return added, removed, nil
}

// GrantEnterpriseFeatures inserts grants that are not already present.
func GrantEnterpriseFeatures(tx *gorm.DB, orgID uuid.UUID, keys []string, source string, grantedBy *uuid.UUID) (int64, error) {
	unique := uniqueFeatureKeys(keys)
	if len(unique) == 0 {
		return 0, nil
	}

	now := time.Now()
	var actor *uuid.UUID
	if grantedBy != nil {
		copyID := *grantedBy
		actor = &copyID
	}
	rows := make([]OrganizationEnterpriseFeature, 0, len(unique))
	for _, key := range unique {
		rows = append(rows, OrganizationEnterpriseFeature{
			OrganizationID: orgID,
			FeatureKey:     key,
			GrantedBy:      actor,
			Source:         source,
			CreatedAt:      now,
		})
	}

	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "feature_key"}},
		DoNothing: true,
	}).Create(&rows)

	return result.RowsAffected, result.Error
}

// PruneEnterpriseFeaturesNotIn deletes grants whose feature is not in keys.
// An empty list deletes every grant.
func PruneEnterpriseFeaturesNotIn(tx *gorm.DB, keys []string) ([]OrganizationEnterpriseFeature, error) {
	unique := uniqueFeatureKeys(keys)
	if len(unique) == 0 {
		return DeleteAllEnterpriseFeatures(tx)
	}

	return deleteEnterpriseFeaturesReturning(tx, `
		DELETE FROM organization_enterprise_features
		WHERE NOT (feature_key = ANY(?::text[]))
		RETURNING organization_id, feature_key, granted_by, source, created_at
	`, pq.Array(unique))
}

func DeleteAllEnterpriseFeatures(tx *gorm.DB) ([]OrganizationEnterpriseFeature, error) {
	return deleteEnterpriseFeaturesReturning(tx, `
		DELETE FROM organization_enterprise_features
		RETURNING organization_id, feature_key, granted_by, source, created_at
	`)
}

func deleteEnterpriseFeaturesReturning(tx *gorm.DB, query string, args ...any) ([]OrganizationEnterpriseFeature, error) {
	var removed []OrganizationEnterpriseFeature
	err := tx.Raw(query, args...).Scan(&removed).Error
	if err != nil {
		return nil, err
	}

	return removed, nil
}

func uniqueFeatureKeys(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(keys))
	unique := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}

	return unique
}

func replaceOrganizationEnterpriseFeatures(tx *gorm.DB, orgID uuid.UUID, keys []string, grantedBy uuid.UUID) (added, removed []OrganizationEnterpriseFeature, err error) {
	current, err := ListOrganizationEnterpriseFeatures(tx, orgID)
	if err != nil {
		return nil, nil, err
	}

	currentByKey := make(map[string]OrganizationEnterpriseFeature, len(current))
	for _, row := range current {
		currentByKey[row.FeatureKey] = row
	}

	wanted := uniqueFeatureKeys(keys)
	wantedSet := make(map[string]struct{}, len(wanted))
	var deleteKeys []string
	for _, key := range wanted {
		wantedSet[key] = struct{}{}
	}
	for _, row := range current {
		if _, ok := wantedSet[row.FeatureKey]; ok {
			continue
		}
		removed = append(removed, row)
		deleteKeys = append(deleteKeys, row.FeatureKey)
	}
	if len(deleteKeys) > 0 {
		err = tx.Where("organization_id = ? AND feature_key IN ?", orgID, deleteKeys).
			Delete(&OrganizationEnterpriseFeature{}).
			Error
		if err != nil {
			return nil, nil, err
		}
	}

	now := time.Now()
	actor := grantedBy
	toInsert := make([]OrganizationEnterpriseFeature, 0)
	for _, key := range wanted {
		if _, ok := currentByKey[key]; ok {
			continue
		}
		toInsert = append(toInsert, OrganizationEnterpriseFeature{
			OrganizationID: orgID,
			FeatureKey:     key,
			GrantedBy:      &actor,
			Source:         EnterpriseFeatureSourceAdmin,
			CreatedAt:      now,
		})
	}
	if len(toInsert) == 0 {
		return nil, removed, nil
	}

	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "feature_key"}},
		DoNothing: true,
	}).Create(&toInsert)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	if result.RowsAffected == int64(len(toInsert)) {
		return toInsert, removed, nil
	}

	latest, err := ListOrganizationEnterpriseFeatures(tx, orgID)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range latest {
		if _, ok := currentByKey[row.FeatureKey]; ok {
			continue
		}
		added = append(added, row)
	}

	return added, removed, nil
}
