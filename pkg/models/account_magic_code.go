package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type AccountMagicCode struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string     `gorm:"type:varchar(255);not null"`
	CodeHash  string     `gorm:"type:varchar(64);not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"default:null"`
	CreatedAt time.Time
}

func (AccountMagicCode) TableName() string {
	return "account_magic_codes"
}

func CreateAccountMagicCode(email, codeHash string, expiresAt time.Time) (*AccountMagicCode, error) {
	return CreateAccountMagicCodeInTransaction(database.Conn(), email, codeHash, expiresAt)
}

func CreateAccountMagicCodeInTransaction(tx *gorm.DB, email, codeHash string, expiresAt time.Time) (*AccountMagicCode, error) {
	code := &AccountMagicCode{
		Email:     email,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
	}

	err := tx.Create(code).Error
	if err != nil {
		return nil, err
	}

	return code, nil
}

func FindValidAccountMagicCode(email, codeHash string) (*AccountMagicCode, error) {
	return FindValidAccountMagicCodeInTransaction(database.Conn(), email, codeHash)
}

func FindValidAccountMagicCodeInTransaction(tx *gorm.DB, email, codeHash string) (*AccountMagicCode, error) {
	var code AccountMagicCode

	err := tx.
		Where("email = ?", email).
		Where("code_hash = ?", codeHash).
		Where("expires_at > ?", time.Now()).
		Where("used_at IS NULL").
		First(&code).
		Error

	if err != nil {
		return nil, err
	}

	return &code, nil
}

func CountRecentMagicCodes(email string, since time.Time) (int64, error) {
	return CountRecentMagicCodesInTransaction(database.Conn(), email, since)
}

func CountRecentMagicCodesInTransaction(tx *gorm.DB, email string, since time.Time) (int64, error) {
	var count int64

	err := tx.Model(&AccountMagicCode{}).
		Where("email = ?", email).
		Where("created_at > ?", since).
		Count(&count).
		Error

	return count, err
}

func (c *AccountMagicCode) MarkUsed() error {
	return c.MarkUsedInTransaction(database.Conn())
}

func (c *AccountMagicCode) MarkUsedInTransaction(tx *gorm.DB) error {
	now := time.Now()
	c.UsedAt = &now
	return tx.Model(c).Update("used_at", now).Error
}
