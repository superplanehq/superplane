package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrSCIMTokenLimit is returned when a connection already has five active tokens.
var ErrSCIMTokenLimit = errors.New("this organization already has five active SCIM tokens")

// SCIMToken is a bearer credential for the SCIM endpoint of one connection.
type SCIMToken struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ConnectionID uuid.UUID
	TokenHash    string
	Description  string
	CreatedBy    *uuid.UUID
	CreatedAt    time.Time
	LastUsedAt   *time.Time
	RevokedAt    *time.Time
}

func (SCIMToken) TableName() string { return "scim_tokens" }

func CreateSCIMToken(tx *gorm.DB, token *SCIMToken) error {
	var connection SSOConnection
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ?", token.ConnectionID).
		First(&connection).Error
	if err != nil {
		return err
	}
	count, err := CountActiveSCIMTokens(tx, token.ConnectionID)
	if err != nil {
		return err
	}
	if count >= SSOMaxActiveSCIMTokens {
		return ErrSCIMTokenLimit
	}
	return tx.Create(token).Error
}

func ListActiveSCIMTokens(tx *gorm.DB, connectionID uuid.UUID) ([]SCIMToken, error) {
	var tokens []SCIMToken
	err := tx.Where("connection_id = ? AND revoked_at IS NULL", connectionID).
		Order("created_at, id").
		Find(&tokens).Error
	return tokens, err
}

func CountActiveSCIMTokens(tx *gorm.DB, connectionID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&SCIMToken{}).
		Where("connection_id = ? AND revoked_at IS NULL", connectionID).
		Count(&count).Error
	return count, err
}

func FindActiveSCIMTokenByHash(tx *gorm.DB, tokenHash string) (*SCIMToken, error) {
	var token SCIMToken
	err := tx.Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func FindActiveSCIMToken(tx *gorm.DB, connectionID, id uuid.UUID) (*SCIMToken, error) {
	var token SCIMToken
	err := tx.Where("connection_id = ? AND id = ? AND revoked_at IS NULL", connectionID, id).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (token *SCIMToken) Revoke(tx *gorm.DB, now time.Time) error {
	token.RevokedAt = &now
	return tx.Model(token).Update("revoked_at", now).Error
}

// TouchSCIMTokenLastUsed records use at most once per minute.
func TouchSCIMTokenLastUsed(tx *gorm.DB, id uuid.UUID, now time.Time) error {
	minuteAgo := now.Add(-time.Minute)
	return tx.Model(&SCIMToken{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", id, minuteAgo).
		Update("last_used_at", now).Error
}
