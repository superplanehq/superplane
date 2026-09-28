package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HostedAppInstallationMember caches whether a provider login can use a
// hosted app installation. GitHub is the source of truth; a row only avoids
// repeated membership lookups, so callers re-check when checked_at is stale.
// Member logins are stored lowercased.
type HostedAppInstallationMember struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Provider       string
	InstallationID string
	MemberLogin    string
	Allowed        bool
	// Errored marks a check that failed instead of answering, for example a
	// missing app permission. Callers retry errored rows sooner than clean
	// answers.
	Errored   bool
	CheckedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (HostedAppInstallationMember) TableName() string {
	return "hosted_app_installation_members"
}

// UpsertHostedAppInstallationMember stores the latest membership check result
// for a login on an installation.
func UpsertHostedAppInstallationMember(tx *gorm.DB, row HostedAppInstallationMember) error {
	row.Provider = strings.TrimSpace(row.Provider)
	row.InstallationID = strings.TrimSpace(row.InstallationID)
	row.MemberLogin = strings.ToLower(strings.TrimSpace(row.MemberLogin))
	if row.Provider == "" || row.InstallationID == "" || row.MemberLogin == "" {
		return nil
	}
	if row.CheckedAt.IsZero() {
		row.CheckedAt = time.Now().UTC()
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}, {Name: "installation_id"}, {Name: "member_login"}},
		DoUpdates: clause.Assignments(map[string]any{
			"allowed":    row.Allowed,
			"errored":    row.Errored,
			"checked_at": row.CheckedAt,
			"updated_at": time.Now().UTC(),
		}),
	}).Create(&row).Error
}

// FindHostedAppInstallationMember returns the cached membership check for a
// login on an installation, or nil when the login was never checked.
func FindHostedAppInstallationMember(tx *gorm.DB, provider, installationID, memberLogin string) (*HostedAppInstallationMember, error) {
	provider = strings.TrimSpace(provider)
	installationID = strings.TrimSpace(installationID)
	memberLogin = strings.ToLower(strings.TrimSpace(memberLogin))
	if provider == "" || installationID == "" || memberLogin == "" {
		return nil, nil
	}

	var row HostedAppInstallationMember
	err := tx.
		Where("provider = ? AND installation_id = ? AND member_login = ?", provider, installationID, memberLogin).
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
