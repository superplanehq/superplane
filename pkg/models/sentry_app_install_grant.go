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
	InstallationUUID     string `gorm:"primaryKey"`
	CodeDigest           string
	OrganizationSlug     string
	AccessToken          []byte
	RefreshToken         []byte
	TokenExpiresAt       string
	ClaimedIntegrationID *string
	ExpiresAt            time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (SentryAppInstallGrant) TableName() string {
	return "sentry_app_install_grants"
}

// ErrSentryAppInstallationDeleted means Sentry already deleted the
// installation, so SuperPlane must not store a grant for it.
var ErrSentryAppInstallationDeleted = errors.New("sentry app installation was deleted")

// UpsertSentryAppInstallGrant replaces the grant of an installation. Sentry
// sends installation.created again after a reinstall, and only the newest
// grant can complete a connection. A second write of the same grant keeps
// the tokens and the claim of the first write when it brings no tokens, so
// a retry cannot erase working credentials. A deleted installation returns
// ErrSentryAppInstallationDeleted and stores nothing.
func UpsertSentryAppInstallGrant(tx *gorm.DB, grant SentryAppInstallGrant) error {
	grant.InstallationUUID = strings.TrimSpace(grant.InstallationUUID)
	if grant.InstallationUUID == "" || grant.CodeDigest == "" {
		return nil
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		if err := LockSentryAppInstallation(inner, grant.InstallationUUID); err != nil {
			return err
		}
		deleted, err := SentryAppInstallationDeleted(inner, grant.InstallationUUID)
		if err != nil {
			return err
		}
		if deleted {
			return ErrSentryAppInstallationDeleted
		}

		sameGrant := "sentry_app_install_grants.code_digest = excluded.code_digest"
		keepWhenEmpty := func(column string) clause.Expr {
			return gorm.Expr(
				"CASE WHEN " + sameGrant + " AND excluded." + column + " IS NULL THEN sentry_app_install_grants." + column + " ELSE excluded." + column + " END",
			)
		}
		return inner.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "installation_uuid"}},
			DoUpdates: clause.Assignments(map[string]any{
				"code_digest":       gorm.Expr("excluded.code_digest"),
				"organization_slug": gorm.Expr("excluded.organization_slug"),
				"access_token":      keepWhenEmpty("access_token"),
				"refresh_token":     keepWhenEmpty("refresh_token"),
				"token_expires_at": gorm.Expr(
					"CASE WHEN " + sameGrant + " AND excluded.access_token IS NULL THEN sentry_app_install_grants.token_expires_at ELSE excluded.token_expires_at END",
				),
				"claimed_integration_id": gorm.Expr(
					"CASE WHEN " + sameGrant + " THEN sentry_app_install_grants.claimed_integration_id ELSE NULL END",
				),
				"expires_at": gorm.Expr("excluded.expires_at"),
				"updated_at": gorm.Expr("excluded.updated_at"),
			}),
		}).Create(&grant).Error
	})
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

// ClaimSentryAppInstallGrant assigns an unexpired grant to one integration.
// The first caller with the matching code wins. The same integration can
// claim again after a failed bind. A different integration cannot.
func ClaimSentryAppInstallGrant(tx *gorm.DB, installationUUID, codeDigest, integrationID string, now time.Time) (*SentryAppInstallGrant, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	codeDigest = strings.TrimSpace(codeDigest)
	integrationID = strings.TrimSpace(integrationID)
	if installationUUID == "" || codeDigest == "" || integrationID == "" {
		return nil, nil
	}

	grants := []SentryAppInstallGrant{}
	err := tx.Raw(
		`UPDATE sentry_app_install_grants
		 SET claimed_integration_id = ?, updated_at = ?
		 WHERE installation_uuid = ?
		   AND code_digest = ?
		   AND expires_at > ?
		   AND (claimed_integration_id IS NULL OR claimed_integration_id = ?)
		 RETURNING *`,
		integrationID,
		now,
		installationUUID,
		codeDigest,
		now,
		integrationID,
	).Scan(&grants).Error
	if err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, nil
	}
	return &grants[0], nil
}

// ForgetSentryAppInstallation records that Sentry deleted an installation
// and drops its grant in one transaction. A grant write that waits for the
// same installation lock then sees the deletion and stores nothing.
func ForgetSentryAppInstallation(tx *gorm.DB, installationUUID string, deletedAt time.Time) error {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" {
		return nil
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		if err := LockSentryAppInstallation(inner, installationUUID); err != nil {
			return err
		}
		if err := RecordSentryAppInstallationDeletion(inner, installationUUID, deletedAt); err != nil {
			return err
		}
		return DeleteSentryAppInstallGrant(inner, installationUUID)
	})
}

// DeleteSentryAppInstallGrant drops the grant of an installation, for example
// after the setup callback binds it.
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
