package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

// SSOIdentity links an account to one organization connection.
type SSOIdentity struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ConnectionID      uuid.UUID
	OrganizationID    uuid.UUID
	AccountID         uuid.UUID
	UserID            *uuid.UUID
	Subject           *string
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

func FindSSOIdentityByIDPSession(tx *gorm.DB, connectionID uuid.UUID, sessionID string) (*SSOIdentity, error) {
	var identity SSOIdentity
	err := tx.Where("connection_id = ? AND idp_session_id = ?", connectionID, sessionID).First(&identity).Error
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
		query = query.Where("lower(scim_user_name) = lower(?)", filterValue)
	case "externalId":
		query = query.Where("lower(external_id) = lower(?)", filterValue)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var identities []SSOIdentity
	err := query.Session(&gorm.Session{}).Order("created_at, id").Offset(offset).Limit(limit).Find(&identities).Error
	return identities, total, err
}
