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
	ID             int    `gorm:"primary_key"`
	InstallationID string `gorm:"type:varchar(64)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func GetInstallationID() (string, error) {
	return GetInstallationIDInTransaction(database.Conn())
}

func GetInstallationIDInTransaction(tx *gorm.DB) (string, error) {
	var metadata InstallationMetadata
	err := tx.Where("id = ?", installationMetadataID).First(&metadata).Error
	if err == nil {
		return metadata.InstallationID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	installationID := uuid.NewString()
	metadata = InstallationMetadata{
		ID:             installationMetadataID,
		InstallationID: installationID,
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&metadata).Error; err != nil {
		return "", err
	}

	if err := tx.Where("id = ?", installationMetadataID).First(&metadata).Error; err != nil {
		return "", err
	}

	return metadata.InstallationID, nil
}
