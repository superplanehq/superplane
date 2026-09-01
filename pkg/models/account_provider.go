package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrProviderLinkedToAnotherAccount = errors.New("provider identity is linked to another account")
	ErrAccountProviderConflict        = errors.New("account is linked to another provider identity")
)

type AccountProvider struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AccountID      uuid.UUID
	Provider       string
	ProviderID     string
	Username       string
	Email          string
	Name           string
	AvatarURL      string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func ListAccountProviders(tx *gorm.DB, accountID uuid.UUID) ([]AccountProvider, error) {
	providers := []AccountProvider{}
	err := tx.Where("account_id = ?", accountID).Order("created_at ASC").Find(&providers).Error
	if err != nil {
		return nil, err
	}

	return providers, nil
}

func SaveLinkedAccountProvider(tx *gorm.DB, provider *AccountProvider) error {
	err := tx.Transaction(func(tx *gorm.DB) error {
		var linkedIdentity AccountProvider
		err := tx.
			Where("provider = ? AND provider_id = ?", provider.Provider, provider.ProviderID).
			First(&linkedIdentity).
			Error
		if err == nil {
			if linkedIdentity.AccountID != provider.AccountID {
				return ErrProviderLinkedToAnotherAccount
			}

			provider.ID = linkedIdentity.ID
			return tx.Model(&linkedIdentity).Updates(accountProviderUpdates(provider)).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var accountIdentity AccountProvider
		err = tx.
			Where("account_id = ? AND provider = ?", provider.AccountID, provider.Provider).
			First(&accountIdentity).
			Error
		if err == nil {
			if accountIdentity.ProviderID != provider.ProviderID {
				return ErrAccountProviderConflict
			}

			provider.ID = accountIdentity.ID
			return tx.Model(&accountIdentity).Updates(accountProviderUpdates(provider)).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return tx.Create(provider).Error
	})
	if err == nil || errors.Is(err, ErrProviderLinkedToAnotherAccount) || errors.Is(err, ErrAccountProviderConflict) {
		return err
	}

	// A concurrent insert can win either unique constraint. Classify the
	// resulting error with a fresh transaction because PostgreSQL aborts the
	// transaction that observed the uniqueness violation.
	var linkedIdentity AccountProvider
	identityErr := tx.
		Where("provider = ? AND provider_id = ?", provider.Provider, provider.ProviderID).
		First(&linkedIdentity).
		Error
	if identityErr == nil {
		if linkedIdentity.AccountID != provider.AccountID {
			return ErrProviderLinkedToAnotherAccount
		}
		return nil
	}

	var accountIdentity AccountProvider
	accountErr := tx.
		Where("account_id = ? AND provider = ?", provider.AccountID, provider.Provider).
		First(&accountIdentity).
		Error
	if accountErr == nil {
		if accountIdentity.ProviderID != provider.ProviderID {
			return ErrAccountProviderConflict
		}
		return nil
	}

	return err
}

func accountProviderUpdates(provider *AccountProvider) map[string]any {
	return map[string]any{
		"username":         provider.Username,
		"email":            provider.Email,
		"name":             provider.Name,
		"avatar_url":       provider.AvatarURL,
		"access_token":     provider.AccessToken,
		"refresh_token":    provider.RefreshToken,
		"token_expires_at": provider.TokenExpiresAt,
	}
}
