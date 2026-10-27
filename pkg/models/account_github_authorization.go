package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AccountGitHubAuthorization is the user access token a member grants to the
// public SuperPlane GitHub App. SuperPlane uses it only to open pull requests
// as that member. A row exists only while the member keeps the option on.
type AccountGitHubAuthorization struct {
	AccountID             uuid.UUID `gorm:"type:uuid;primary_key"`
	GitHubAppID           int64     `gorm:"column:github_app_id"`
	GitHubUserID          string    `gorm:"column:github_user_id"`
	GitHubLogin           string    `gorm:"column:github_login"`
	EncryptedAccessToken  []byte
	AccessTokenExpiresAt  *time.Time
	EncryptedRefreshToken []byte
	RefreshTokenExpiresAt *time.Time
	InvalidatedAt         *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (a *AccountGitHubAuthorization) TableName() string {
	return "account_github_authorizations"
}

// UsableFor reports whether the token belongs to the app and still works.
func (a *AccountGitHubAuthorization) UsableFor(appID int64) bool {
	return a.InvalidatedAt == nil && a.GitHubAppID == appID
}

func (a *AccountGitHubAuthorization) UpdateTokens(
	tx *gorm.DB,
	encryptedAccessToken []byte,
	accessTokenExpiresAt *time.Time,
	encryptedRefreshToken []byte,
	refreshTokenExpiresAt *time.Time,
) error {
	if encryptedRefreshToken == nil {
		encryptedRefreshToken = []byte{}
	}
	now := time.Now()
	err := tx.Model(a).Updates(map[string]any{
		"encrypted_access_token":   encryptedAccessToken,
		"access_token_expires_at":  accessTokenExpiresAt,
		"encrypted_refresh_token":  encryptedRefreshToken,
		"refresh_token_expires_at": refreshTokenExpiresAt,
		"updated_at":               now,
	}).Error
	if err != nil {
		return err
	}
	a.EncryptedAccessToken = encryptedAccessToken
	a.AccessTokenExpiresAt = accessTokenExpiresAt
	a.EncryptedRefreshToken = encryptedRefreshToken
	a.RefreshTokenExpiresAt = refreshTokenExpiresAt
	a.UpdatedAt = now
	return nil
}

// Invalidate keeps the row so the member sees that the option needs a new
// authorization, but SuperPlane stops using the token.
func (a *AccountGitHubAuthorization) Invalidate(tx *gorm.DB) error {
	now := time.Now()
	err := tx.Model(a).Updates(map[string]any{
		"invalidated_at": now,
		"updated_at":     now,
	}).Error
	if err != nil {
		return err
	}
	a.InvalidatedAt = &now
	a.UpdatedAt = now
	return nil
}

func FindAccountGitHubAuthorization(tx *gorm.DB, accountID uuid.UUID) (*AccountGitHubAuthorization, error) {
	var authorization AccountGitHubAuthorization
	err := tx.Where("account_id = ?", accountID).First(&authorization).Error
	if err != nil {
		return nil, err
	}
	return &authorization, nil
}

// LockAccountGitHubAuthorization locks the row until the transaction ends.
// GitHub refresh tokens are single use, so only one caller may refresh.
func LockAccountGitHubAuthorization(tx *gorm.DB, accountID uuid.UUID) (*AccountGitHubAuthorization, error) {
	var authorization AccountGitHubAuthorization
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("account_id = ?", accountID).
		First(&authorization).
		Error
	if err != nil {
		return nil, err
	}
	return &authorization, nil
}

// SaveAccountGitHubAuthorization stores a new grant and clears an earlier
// invalidation.
func SaveAccountGitHubAuthorization(tx *gorm.DB, authorization *AccountGitHubAuthorization) error {
	now := time.Now()
	authorization.InvalidatedAt = nil
	authorization.CreatedAt = now
	authorization.UpdatedAt = now
	if authorization.EncryptedRefreshToken == nil {
		authorization.EncryptedRefreshToken = []byte{}
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "account_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"github_app_id",
			"github_user_id",
			"github_login",
			"encrypted_access_token",
			"access_token_expires_at",
			"encrypted_refresh_token",
			"refresh_token_expires_at",
			"invalidated_at",
			"updated_at",
		}),
	}).Create(authorization).Error
}

func (a *AccountGitHubAuthorization) Delete(tx *gorm.DB) error {
	return tx.Where("account_id = ?", a.AccountID).Delete(&AccountGitHubAuthorization{}).Error
}
