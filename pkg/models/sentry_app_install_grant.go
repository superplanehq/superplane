package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SentryAppInstallGrant holds a public Sentry app install that the
// installation webhook received before the browser callback arrived. Sentry
// accepts the install code one time only, so the process that answers the
// callback must read the result of that exchange from here.
type SentryAppInstallGrant struct {
	InstallationUUID string `gorm:"primaryKey"`
	CodeDigest       string
	OrganizationSlug string
	AccessToken      []byte
	RefreshToken     []byte
	TokenExpiresAt   string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (SentryAppInstallGrant) TableName() string {
	return "sentry_app_install_grants"
}

// UpsertSentryAppInstallGrant replaces the grant of an installation. Sentry
// sends installation.created again after a reinstall, and only the newest
// grant can complete a connection.
func UpsertSentryAppInstallGrant(tx *gorm.DB, grant SentryAppInstallGrant) error {
	grant.InstallationUUID = strings.TrimSpace(grant.InstallationUUID)
	if grant.InstallationUUID == "" || grant.CodeDigest == "" {
		return nil
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "installation_uuid"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"code_digest", "organization_slug", "access_token", "refresh_token",
			"token_expires_at", "expires_at", "updated_at",
		}),
	}).Create(&grant).Error
}

// FindSentryAppInstallGrant returns the grant of an installation when the
// caller presents the same install code. It leaves the row in place so a
// later callback can retry if decrypting the tokens or binding the
// integration fails. It returns nil when no unexpired grant matches, so a
// caller without the code cannot claim an installation it does not own.
func FindSentryAppInstallGrant(tx *gorm.DB, installationUUID, codeDigest string, now time.Time) (*SentryAppInstallGrant, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" || codeDigest == "" {
		return nil, nil
	}

	var grant SentryAppInstallGrant
	err := tx.
		Where(
			"installation_uuid = ? AND code_digest = ? AND expires_at > ?",
			installationUUID,
			codeDigest,
			now,
		).
		First(&grant).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &grant, nil
}

// DeleteSentryAppInstallGrant drops the grant of an installation, for example
// after Sentry reports that the organization uninstalled the app.
func DeleteSentryAppInstallGrant(tx *gorm.DB, installationUUID string) error {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" {
		return nil
	}

	return tx.
		Where("installation_uuid = ?", installationUUID).
		Delete(&SentryAppInstallGrant{}).
		Error
}

// DeleteExpiredSentryAppInstallGrants drops grants that no callback claimed in
// time, so install codes and tokens do not stay in the database.
func DeleteExpiredSentryAppInstallGrants(tx *gorm.DB, now time.Time) error {
	return tx.
		Where("expires_at <= ?", now).
		Delete(&SentryAppInstallGrant{}).
		Error
}
