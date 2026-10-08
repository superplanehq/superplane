package models

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationGitHubAppID = 1

// InstallationGitHubApp stores the public SuperPlane GitHub App for a
// self-hosted installation. There is at most one row. Cloud keeps the same
// credentials in process environment instead of this table.
type InstallationGitHubApp struct {
	ID                     int `gorm:"primary_key"`
	GitHubAppID            int64
	Slug                   string
	EncryptedPrivateKey    []byte
	EncryptedWebhookSecret []byte
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (InstallationGitHubApp) TableName() string {
	return "installation_github_apps"
}

func FindInstallationGitHubApp(tx *gorm.DB) (*InstallationGitHubApp, error) {
	var app InstallationGitHubApp
	err := tx.Where("id = ?", installationGitHubAppID).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func SaveInstallationGitHubApp(
	tx *gorm.DB,
	githubAppID int64,
	slug string,
	encryptedPrivateKey []byte,
	encryptedWebhookSecret []byte,
) error {
	now := time.Now()
	app := InstallationGitHubApp{
		ID:                     installationGitHubAppID,
		GitHubAppID:            githubAppID,
		Slug:                   slug,
		EncryptedPrivateKey:    encryptedPrivateKey,
		EncryptedWebhookSecret: encryptedWebhookSecret,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"github_app_id",
			"slug",
			"encrypted_private_key",
			"encrypted_webhook_secret",
			"updated_at",
		}),
	}).Create(&app).Error
}
