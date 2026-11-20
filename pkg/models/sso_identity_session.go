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

func (identity *SSOIdentity) RememberIDPSession(tx *gorm.DB, sessionID string, signedInAt, expiresAt time.Time) error {
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
		CreatedAt:    signedInAt,
	}).Error
}

// FindUnrevokedSSOIdentitySession loads a provider session that logout can
// still match. signedInAfter is the oldest sign-in whose account cookie can
// remain. The caller applies the connection's current session rules.
func FindUnrevokedSSOIdentitySession(tx *gorm.DB, connectionID uuid.UUID, sessionID string, signedInAfter time.Time) (*SSOIdentitySession, error) {
	var session SSOIdentitySession
	err := tx.Where(
		"connection_id = ? AND idp_session_id = ? AND revoked_at IS NULL AND created_at >= ?",
		connectionID,
		sessionID,
		signedInAfter,
	).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteUnusedSSOIdentitySessions removes rows that can no longer end access.
// A revoked row is finished. A sign-in older than unusedBefore belongs to an
// account cookie that can no longer exist. The delete is limited.
func DeleteUnusedSSOIdentitySessions(tx *gorm.DB, unusedBefore time.Time, limit int) (int64, error) {
	return deleteRowsLimited(tx, &SSOIdentitySession{}, limit, "revoked_at IS NOT NULL OR created_at < ?", unusedBefore)
}

func (identity *SSOIdentity) RevokeIDPSessions(tx *gorm.DB, at time.Time) error {
	return tx.Model(&SSOIdentitySession{}).
		Where("connection_id = ? AND identity_id = ? AND revoked_at IS NULL", identity.ConnectionID, identity.ID).
		Update("revoked_at", at).Error
}
