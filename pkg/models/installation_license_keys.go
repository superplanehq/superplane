package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationLicenseKeysID = 1

// InstallationLicenseKeys caches the newest root-signed list of license
// signing keys. The document is public and signed, so it is not encrypted,
// and readers verify it before they trust it. There is at most one row.
type InstallationLicenseKeys struct {
	ID        int `gorm:"primary_key"`
	Document  string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (InstallationLicenseKeys) TableName() string {
	return "installation_license_keys"
}

// FindInstallationLicenseKeys returns gorm.ErrRecordNotFound when no key list
// is cached.
func FindInstallationLicenseKeys(tx *gorm.DB) (*InstallationLicenseKeys, error) {
	var keys InstallationLicenseKeys
	err := tx.Where("id = ?", installationLicenseKeysID).First(&keys).Error
	if err != nil {
		return nil, err
	}

	return &keys, nil
}

// SaveInstallationLicenseKeys stores a key list only when its version is
// newer than the cached one.
func SaveInstallationLicenseKeys(tx *gorm.DB, document string, version int64) error {
	now := time.Now()
	keys := InstallationLicenseKeys{
		ID:        installationLicenseKeysID,
		Document:  document,
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"document", "version", "updated_at"}),
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "installation_license_keys.version < EXCLUDED.version"},
		}},
	}).Create(&keys).Error
}
