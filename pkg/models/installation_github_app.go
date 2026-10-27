package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationGitHubAppID = 1

var ErrInstallationGitHubAppExists = errors.New("installation GitHub App already exists")

// InstallationGitHubApp stores the public SuperPlane GitHub App for a
// self-hosted installation. There is at most one row. Cloud keeps the same
// credentials in process environment instead of this table.
type InstallationGitHubApp struct {
	ID                     int   `gorm:"primary_key"`
	GitHubAppID            int64 `gorm:"column:github_app_id"`
	Slug                   string
	EncryptedPrivateKey    []byte
	EncryptedWebhookSecret []byte
	ClientID               string
	EncryptedClientSecret  []byte
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

func SaveInstallationGitHubApp(tx *gorm.DB, app InstallationGitHubApp) error {
	now := time.Now()
	app.ID = installationGitHubAppID
	app.CreatedAt = now
	app.UpdatedAt = now
	if app.EncryptedClientSecret == nil {
		app.EncryptedClientSecret = []byte{}
	}

	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&app)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInstallationGitHubAppExists
	}
	return nil
}
