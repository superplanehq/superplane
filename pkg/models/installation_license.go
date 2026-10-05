package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationLicenseID = 1

// InstallationLicense stores the encrypted Enterprise license for the whole
// installation. There is at most one row.
type InstallationLicense struct {
	ID               int `gorm:"primary_key"`
	EncryptedLicense []byte
	InstalledBy      *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (InstallationLicense) TableName() string {
	return "installation_licenses"
}

// FindInstallationLicense returns gorm.ErrRecordNotFound when no license is installed.
func FindInstallationLicense(tx *gorm.DB) (*InstallationLicense, error) {
	var license InstallationLicense
	err := tx.Where("id = ?", installationLicenseID).First(&license).Error
	if err != nil {
		return nil, err
	}

	return &license, nil
}

func SaveInstallationLicense(tx *gorm.DB, encryptedLicense []byte, installedBy *uuid.UUID) error {
	now := time.Now()
	license := InstallationLicense{
		ID:               installationLicenseID,
		EncryptedLicense: encryptedLicense,
		InstalledBy:      installedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"encrypted_license", "installed_by", "updated_at"}),
	}).Create(&license).Error
}

func DeleteInstallationLicense(tx *gorm.DB) error {
	return tx.Where("id = ?", installationLicenseID).Delete(&InstallationLicense{}).Error
}
