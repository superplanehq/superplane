package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationMetadataID = 1

type InstallationMetadata struct {
	ID                        int    `gorm:"primary_key"`
	InstallationID            string `gorm:"type:varchar(64)"`
	AllowPrivateNetworkAccess bool
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func GetInstallationMetadata() (*InstallationMetadata, error) {
	return GetInstallationMetadataInTransaction(database.Conn())
}

func GetInstallationID() (string, error) {
	return GetInstallationIDInTransaction(database.Conn())
}

func GetInstallationMetadataInTransaction(tx *gorm.DB) (*InstallationMetadata, error) {
	return findOrCreateInstallationMetadataInTransaction(tx)
}

func GetInstallationIDInTransaction(tx *gorm.DB) (string, error) {
	metadata, err := findOrCreateInstallationMetadataInTransaction(tx)
	if err != nil {
		return "", err
	}

	return metadata.InstallationID, nil
}

func UpdateInstallationMetadata(metadata *InstallationMetadata) error {
	return UpdateInstallationMetadataInTransaction(database.Conn(), metadata)
}

func UpdateInstallationMetadataInTransaction(tx *gorm.DB, metadata *InstallationMetadata) error {
	if _, err := findOrCreateInstallationMetadataInTransaction(tx); err != nil {
		return err
	}

	return tx.Model(&InstallationMetadata{}).
		Where("id = ?", installationMetadataID).
		Updates(map[string]any{
			"allow_private_network_access": metadata.AllowPrivateNetworkAccess,
			"updated_at":                   metadata.UpdatedAt,
		}).
		Error
}

func findOrCreateInstallationMetadataInTransaction(tx *gorm.DB) (*InstallationMetadata, error) {
	var metadata InstallationMetadata
	err := tx.Where("id = ?", installationMetadataID).First(&metadata).Error
	if err == nil {
		return &metadata, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	metadata = InstallationMetadata{
		ID:             installationMetadataID,
		InstallationID: uuid.NewString(),
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&metadata).Error; err != nil {
		return nil, err
	}

	if err := tx.Where("id = ?", installationMetadataID).First(&metadata).Error; err != nil {
		return nil, err
	}

	return &metadata, nil
}
