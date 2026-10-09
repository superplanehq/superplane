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
	ClientID               string `gorm:"column:client_id"`
	EncryptedPrivateKey    []byte
	EncryptedWebhookSecret []byte
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

func SaveInstallationGitHubApp(
	tx *gorm.DB,
	githubAppID int64,
	slug string,
	clientID string,
	encryptedPrivateKey []byte,
	encryptedWebhookSecret []byte,
	encryptedClientSecret []byte,
) error {
	now := time.Now()
	app := InstallationGitHubApp{
		ID:                     installationGitHubAppID,
		GitHubAppID:            githubAppID,
		Slug:                   slug,
		ClientID:               clientID,
		EncryptedPrivateKey:    encryptedPrivateKey,
		EncryptedWebhookSecret: encryptedWebhookSecret,
		EncryptedClientSecret:  encryptedClientSecret,
		CreatedAt:              now,
		UpdatedAt:              now,
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

// ReplaceInstallationGitHubAppOAuth stores login credentials on the app that
// was created before those credentials were kept. It updates only the client
// id and secret, and only when the GitHub App id matches. A different app,
// or a row that already has a client id, is left unchanged.
func ReplaceInstallationGitHubAppOAuth(
	tx *gorm.DB,
	githubAppID int64,
	clientID string,
	encryptedClientSecret []byte,
) error {
	result := tx.Model(&InstallationGitHubApp{}).
		Where("id = ? AND client_id = '' AND github_app_id = ?", installationGitHubAppID, githubAppID).
		Updates(map[string]any{
			"client_id":               clientID,
			"encrypted_client_secret": encryptedClientSecret,
			"updated_at":              time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInstallationGitHubAppExists
	}
	return nil
}

// UpdateInstallationGitHubAppLoginClient replaces the login client on the
// saved GitHub App. It does not change the app id, private key, or webhook
// secret, so an administrator can correct a client id or secret.
func UpdateInstallationGitHubAppLoginClient(
	tx *gorm.DB,
	githubAppID int64,
	clientID string,
	encryptedClientSecret []byte,
) error {
	result := tx.Model(&InstallationGitHubApp{}).
		Where("id = ? AND github_app_id = ?", installationGitHubAppID, githubAppID).
		Updates(map[string]any{
			"client_id":               clientID,
			"encrypted_client_secret": encryptedClientSecret,
			"updated_at":              time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInstallationGitHubAppExists
	}
	return nil
}
