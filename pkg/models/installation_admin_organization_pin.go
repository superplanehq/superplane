package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InstallationAdminOrganizationPin struct {
	AccountID      uuid.UUID `gorm:"primaryKey"`
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	PinnedAt       time.Time
	CreatedAt      time.Time
}

func (InstallationAdminOrganizationPin) TableName() string {
	return "installation_admin_organization_pins"
}

func PinOrganizationForAccount(tx *gorm.DB, accountID, organizationID uuid.UUID) error {
	now := time.Now()
	pin := InstallationAdminOrganizationPin{
		AccountID:      accountID,
		OrganizationID: organizationID,
		PinnedAt:       now,
		CreatedAt:      now,
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "account_id"},
			{Name: "organization_id"},
		},
		DoNothing: true,
	}).Create(&pin).Error
}

func UnpinOrganizationForAccount(tx *gorm.DB, accountID, organizationID uuid.UUID) error {
	return tx.
		Where("account_id = ? AND organization_id = ?", accountID, organizationID).
		Delete(&InstallationAdminOrganizationPin{}).
		Error
}

func IsOrganizationPinnedForAccount(tx *gorm.DB, accountID, organizationID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&InstallationAdminOrganizationPin{}).
		Where("account_id = ? AND organization_id = ?", accountID, organizationID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func ListPinnedOrganizationsForAccount(tx *gorm.DB, accountID uuid.UUID) ([]OrganizationWithCounts, error) {
	query := withOrganizationCounts(tx, tx.Model(&Organization{}).
		Joins(
			"JOIN installation_admin_organization_pins ON installation_admin_organization_pins.organization_id = organizations.id AND installation_admin_organization_pins.account_id = ?",
			accountID,
		).
		Where("organizations.deleted_at IS NULL"),
	)

	var organizations []OrganizationWithCounts
	err := query.
		Order("installation_admin_organization_pins.pinned_at DESC, organizations.name ASC").
		Find(&organizations).Error
	if err != nil {
		return nil, err
	}

	return organizations, nil
}
