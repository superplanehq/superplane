package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserAPIToken is a single named personal API token belonging to a human
// user. A user can hold several tokens at once; each authenticates as the
// owning user. Revoking a token hard-deletes its row so the other tokens
// belonging to the same user keep working.
type UserAPIToken struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID
	Name       string
	TokenHash  string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

func NewUserAPIToken(userID uuid.UUID, name, tokenHash string) *UserAPIToken {
	return &UserAPIToken{
		UserID:    userID,
		Name:      name,
		TokenHash: tokenHash,
	}
}

func (t *UserAPIToken) TableName() string {
	return "user_api_tokens"
}

func CreateUserAPIToken(tx *gorm.DB, token *UserAPIToken) error {
	return tx.Create(token).Error
}

func ListUserAPITokens(tx *gorm.DB, userID uuid.UUID) ([]UserAPIToken, error) {
	var tokens []UserAPIToken
	err := tx.
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&tokens).
		Error

	return tokens, err
}

func FindUserAPITokenByHash(tx *gorm.DB, tokenHash string) (*UserAPIToken, error) {
	var token UserAPIToken
	err := tx.
		Where("token_hash = ?", tokenHash).
		First(&token).
		Error
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func FindUserAPIToken(tx *gorm.DB, userID, id uuid.UUID) (*UserAPIToken, error) {
	var token UserAPIToken
	err := tx.
		Where("user_id = ?", userID).
		Where("id = ?", id).
		First(&token).
		Error
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func CountUserAPITokens(tx *gorm.DB, userID uuid.UUID) (int64, error) {
	var count int64
	err := tx.
		Model(&UserAPIToken{}).
		Where("user_id = ?", userID).
		Count(&count).
		Error

	return count, err
}

func TouchUserAPITokenLastUsed(tx *gorm.DB, id uuid.UUID, when time.Time) error {
	return tx.
		Model(&UserAPIToken{}).
		Where("id = ?", id).
		Update("last_used_at", when).
		Error
}

// HardDelete permanently removes the token so it stops authenticating
// immediately. Other tokens belonging to the same user are unaffected.
func (t *UserAPIToken) HardDelete(tx *gorm.DB) error {
	return tx.Delete(t).Error
}

// DeleteUserAPITokensForAccount deletes every personal API token belonging
// to any user of the given account. It mirrors
// ClearAPIKeyTokenHashesCreatedByAccount so that a password change or an
// account block invalidates every personal token, not just the legacy
// users.token_hash column.
func DeleteUserAPITokensForAccount(tx *gorm.DB, accountID uuid.UUID) error {
	userIDs := tx.Unscoped().Model(&User{}).Select("id").Where("account_id = ?", accountID)
	return tx.
		Where("user_id IN (?)", userIDs).
		Delete(&UserAPIToken{}).
		Error
}
