package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SSOIdentitySession is one identity-provider session that can still end
// SuperPlane access for an identity.
type SSOIdentitySession struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ConnectionID uuid.UUID
	IdentityID   uuid.UUID
	IDPSessionID string `gorm:"column:idp_session_id"`
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

func (SSOIdentitySession) TableName() string { return "sso_identity_sessions" }

func (identity *SSOIdentity) RememberIDPSession(tx *gorm.DB, sessionID string, expiresAt time.Time) error {
	if sessionID == "" {
		return nil
	}
	err := tx.Where("connection_id = ? AND idp_session_id = ? AND revoked_at IS NULL", identity.ConnectionID, sessionID).
		Delete(&SSOIdentitySession{}).Error
	if err != nil {
		return err
	}
	return tx.Create(&SSOIdentitySession{
		ID:           uuid.New(),
		ConnectionID: identity.ConnectionID,
		IdentityID:   identity.ID,
		IDPSessionID: sessionID,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}).Error
}

func (identity *SSOIdentity) RevokeIDPSessions(tx *gorm.DB, at time.Time) error {
	return tx.Model(&SSOIdentitySession{}).
		Where("connection_id = ? AND identity_id = ? AND revoked_at IS NULL", identity.ConnectionID, identity.ID).
		Update("revoked_at", at).Error
}
