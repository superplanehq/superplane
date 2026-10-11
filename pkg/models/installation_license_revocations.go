package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationLicenseRevocationsID = 1

// InstallationLicenseRevocations caches the newest root-signed list of
// revoked license ids. The document is public and signed, so it is not
// encrypted, and readers verify it before they trust it. There is at most
// one row.
type InstallationLicenseRevocations struct {
	ID        int `gorm:"primary_key"`
	Document  string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (InstallationLicenseRevocations) TableName() string {
	return "installation_license_revocations"
}

// FindInstallationLicenseRevocations returns gorm.ErrRecordNotFound when no
// revocation list is cached.
func FindInstallationLicenseRevocations(tx *gorm.DB) (*InstallationLicenseRevocations, error) {
	var list InstallationLicenseRevocations
	err := tx.Where("id = ?", installationLicenseRevocationsID).First(&list).Error
	if err != nil {
		return nil, err
	}

	return &list, nil
}

// SaveInstallationLicenseRevocations stores a revocation list only when its
// version is newer than the cached one.
func SaveInstallationLicenseRevocations(tx *gorm.DB, document string, version int64) error {
	now := time.Now()
	list := InstallationLicenseRevocations{
		ID:        installationLicenseRevocationsID,
		Document:  document,
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"document", "version", "updated_at"}),
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "installation_license_revocations.version < EXCLUDED.version"},
		}},
	}).Create(&list).Error
}
