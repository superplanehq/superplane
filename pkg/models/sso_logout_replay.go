package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SSOLogoutReplay stores a Global Token Revocation token id until it expires.
type SSOLogoutReplay struct {
	ConnectionID uuid.UUID `gorm:"primaryKey"`
	JTI          string    `gorm:"primaryKey;column:jti"`
	ExpiresAt    time.Time
}

func (SSOLogoutReplay) TableName() string { return "sso_logout_replays" }

// ConsumeSSOLogoutReplay stores a logout token id until it expires.
// It returns false when that id was already stored for the connection.
func ConsumeSSOLogoutReplay(tx *gorm.DB, connectionID uuid.UUID, jti string, expiresAt time.Time) (bool, error) {
	if err := tx.Where("expires_at <= ?", time.Now()).Delete(&SSOLogoutReplay{}).Error; err != nil {
		return false, err
	}
	err := tx.Create(&SSOLogoutReplay{
		ConnectionID: connectionID,
		JTI:          jti,
		ExpiresAt:    expiresAt,
	}).Error
	if err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// DeleteExpiredSSOLogoutReplays removes token ids that can no longer be replayed.
func DeleteExpiredSSOLogoutReplays(tx *gorm.DB, now time.Time) (int64, error) {
	result := tx.Where("expires_at <= ?", now).Delete(&SSOLogoutReplay{})
	return result.RowsAffected, result.Error
}
