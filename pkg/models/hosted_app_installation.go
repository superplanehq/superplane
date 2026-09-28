package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const HostedAppProviderGitHub = "github"

// HostedAppInstallation is a public SuperPlane app installation recorded
// from a signed webhook. Claim and request adoption read this table instead
// of trusting the browser callback. Provider names the third-party app
// (github today; other hosted apps later).
type HostedAppInstallation struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Provider       string
	InstallationID string
	AccountLogin   string
	AccountType    string
	AccountID      int64
	SenderLogin    string
	LastEventAt    time.Time
	CreatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (HostedAppInstallation) TableName() string {
	return "hosted_app_installations"
}

// UpsertHostedAppInstallation records or refreshes an installation from a
// signed webhook. A reinstall clears a previous soft-delete.
func UpsertHostedAppInstallation(tx *gorm.DB, row HostedAppInstallation) error {
	row.Provider = strings.TrimSpace(row.Provider)
	row.InstallationID = strings.TrimSpace(row.InstallationID)
	if row.Provider == "" || row.InstallationID == "" {
		return nil
	}

	row.AccountLogin = strings.TrimSpace(row.AccountLogin)
	row.AccountType = strings.TrimSpace(row.AccountType)
	row.SenderLogin = strings.TrimSpace(row.SenderLogin)
	if row.LastEventAt.IsZero() {
		row.LastEventAt = time.Now().UTC()
	}

	return tx.Unscoped().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "installation_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"account_login": row.AccountLogin,
			"account_type":  row.AccountType,
			"account_id":    row.AccountID,
			"sender_login":  row.SenderLogin,
			"last_event_at": row.LastEventAt,
			"deleted_at":    nil,
		}),
	}).Create(&row).Error
}

// ReconcileHostedAppInstallation records an installation found through the
// provider's list API. Unlike the webhook upsert it never advances
// last_event_at on an existing row, so a reconciled old installation stays
// outside the no-identity first-claim window. Only signed webhooks freshen
// rows.
func ReconcileHostedAppInstallation(tx *gorm.DB, row HostedAppInstallation) error {
	row.Provider = strings.TrimSpace(row.Provider)
	row.InstallationID = strings.TrimSpace(row.InstallationID)
	if row.Provider == "" || row.InstallationID == "" {
		return nil
	}

	row.AccountLogin = strings.TrimSpace(row.AccountLogin)
	row.AccountType = strings.TrimSpace(row.AccountType)
	if row.LastEventAt.IsZero() {
		row.LastEventAt = row.CreatedAt
	}
	if row.LastEventAt.IsZero() {
		// A missing provider timestamp must not look fresh: the epoch keeps
		// the row outside the no-identity first-claim window.
		row.LastEventAt = time.Unix(0, 0).UTC()
	}

	return tx.Unscoped().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "installation_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"account_login": row.AccountLogin,
			"account_type":  row.AccountType,
			"account_id":    row.AccountID,
			"deleted_at":    nil,
		}),
	}).Create(&row).Error
}

// ListHostedAppInstallations returns every live installation for a provider,
// for identity-based discovery.
func ListHostedAppInstallations(tx *gorm.DB, provider string) ([]HostedAppInstallation, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, nil
	}

	rows := []HostedAppInstallation{}
	err := tx.
		Where("provider = ?", provider).
		Order("account_login ASC").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// TouchHostedAppInstallation refreshes last_event_at for a live installation,
// for example after an installation_repositories webhook.
func TouchHostedAppInstallation(tx *gorm.DB, provider, installationID string, at time.Time) error {
	provider = strings.TrimSpace(provider)
	installationID = strings.TrimSpace(installationID)
	if provider == "" || installationID == "" {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	return tx.
		Model(&HostedAppInstallation{}).
		Where("provider = ? AND installation_id = ?", provider, installationID).
		Update("last_event_at", at).
		Error
}

// FindHostedAppInstallation returns the row for a provider installation id,
// including a soft-deleted row so a claim can reject an uninstall.
func FindHostedAppInstallation(tx *gorm.DB, provider, installationID string) (*HostedAppInstallation, error) {
	provider = strings.TrimSpace(provider)
	installationID = strings.TrimSpace(installationID)
	if provider == "" || installationID == "" {
		return nil, nil
	}

	var row HostedAppInstallation
	err := tx.Unscoped().
		Where("provider = ? AND installation_id = ?", provider, installationID).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindHostedAppInstallationByAccountLogin returns the live installation for a
// provider account login. Used when Sync adopts an approved install request.
func FindHostedAppInstallationByAccountLogin(tx *gorm.DB, provider, accountLogin string) (*HostedAppInstallation, error) {
	provider = strings.TrimSpace(provider)
	accountLogin = strings.TrimSpace(accountLogin)
	if provider == "" || accountLogin == "" {
		return nil, nil
	}

	var row HostedAppInstallation
	err := tx.
		Where("provider = ? AND LOWER(account_login) = ?", provider, strings.ToLower(accountLogin)).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// SoftDeleteHostedAppInstallation marks an installation as uninstalled.
func SoftDeleteHostedAppInstallation(tx *gorm.DB, provider, installationID string) error {
	provider = strings.TrimSpace(provider)
	installationID = strings.TrimSpace(installationID)
	if provider == "" || installationID == "" {
		return nil
	}

	return tx.
		Where("provider = ? AND installation_id = ?", provider, installationID).
		Delete(&HostedAppInstallation{}).
		Error
}
