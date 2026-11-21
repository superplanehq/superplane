package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SSOIdentity links an account to one organization connection.
type SSOIdentity struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ConnectionID      uuid.UUID
	OrganizationID    uuid.UUID
	AccountID         uuid.UUID
	UserID            *uuid.UUID
	Subject           *string
	IDPSubject        *string `gorm:"column:idp_subject"`
	ExternalID        *string
	SCIMUserName      *string
	Email             string
	Name              string
	Active            bool
	ProvisionedBy     string
	IDPSessionID      *string `gorm:"column:idp_session_id"`
	IDToken           []byte
	LastLoginAt       *time.Time
	SessionsRevokedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (SSOIdentity) TableName() string { return "sso_identities" }

func (identity *SSOIdentity) Save(tx *gorm.DB) error {
	identity.UpdatedAt = time.Now()
	return tx.Save(identity).Error
}

func (identity *SSOIdentity) RevokeSessions(tx *gorm.DB, at time.Time) error {
	if err := identity.RevokeIDPSessions(tx, at); err != nil {
		return err
	}
	return tx.Model(&SSOIdentity{}).
		Where("connection_id = ? AND id = ?", identity.ConnectionID, identity.ID).
		Updates(map[string]any{
			"sessions_revoked_at": at,
			"updated_at":          time.Now(),
		}).Error
}

func LockSSOIdentity(tx *gorm.DB, connectionID, id uuid.UUID) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("connection_id = ? AND id = ?", connectionID, id).
		First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityBySubject(tx *gorm.DB, connectionID uuid.UUID, subject string) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where("connection_id = ? AND subject = ?", connectionID, subject).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityByPendingEmail(tx *gorm.DB, connectionID uuid.UUID, email string) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where(
		"connection_id = ? AND subject IS NULL AND lower(email) = ?",
		connectionID,
		utils.NormalizeEmail(email),
	).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityByAccount(tx *gorm.DB, connectionID, accountID uuid.UUID) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where("connection_id = ? AND account_id = ?", connectionID, accountID).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityByID(tx *gorm.DB, connectionID, id uuid.UUID) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where("connection_id = ? AND id = ?", connectionID, id).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityByExternalID(tx *gorm.DB, connectionID uuid.UUID, externalID string) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where("connection_id = ? AND external_id = ?", connectionID, externalID).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityByUserName(tx *gorm.DB, connectionID uuid.UUID, userName string) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where(
		"connection_id = ? AND lower(scim_user_name) = lower(?)",
		connectionID,
		userName,
	).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func FindSSOIdentityByIDPSubject(tx *gorm.DB, connectionID uuid.UUID, subject string) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where("connection_id = ? AND idp_subject = ?", connectionID, subject).First(&identity).Error
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

// ListSSOIdentities returns one page of identities for a connection.
// filterAttr is "userName" or "externalId". An empty filter lists every identity.
func ListSSOIdentities(tx *gorm.DB, connectionID uuid.UUID, filterAttr, filterValue string, offset, limit int) ([]SSOIdentity, int64, error) {
	query := tx.Model(&SSOIdentity{}).Where("connection_id = ?", connectionID)
	switch filterAttr {
	case "userName":
		// Identities created by just-in-time sign-in have no SCIM userName yet.
		// The email is the userName Okta looks up before it creates the user.
		query = query.Where("lower(COALESCE(NULLIF(scim_user_name, ''), email)) = lower(?)", filterValue)
	case "externalId":
		query = query.Where("external_id = ?", filterValue)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var identities []SSOIdentity
	err := query.Session(&gorm.Session{}).Order("created_at, id").Offset(offset).Limit(limit).Find(&identities).Error
	return identities, total, err
}
