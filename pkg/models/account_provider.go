package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type AccountProvider struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AccountID  uuid.UUID
	Provider   string
	ProviderID string
	Username   string
	Email      string
	Name       string
	AvatarURL  string

	//
	// TODO: Do we need these?
	//
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func FindAccountProviderByID(id string) (*AccountProvider, error) {
	var account AccountProvider
	accountUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	err = database.Conn().
		Where("id = ?", accountUUID).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, err
}

func (a *Account) FindAccountProviderByID(provider, providerID string) (*AccountProvider, error) {
	var account AccountProvider

	err := database.Conn().
		Where("provider = ?", provider).
		Where("provider_id = ?", providerID).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, err
}

func (a *Account) FindAccountProviders() ([]AccountProvider, error) {
	var accounts []AccountProvider

	err := database.Conn().
		Where("account_id = ?", a.ID).
		Find(&accounts).
		Error

	if err != nil {
		return nil, err
	}

	return accounts, err
}

func (a *Account) FindAccountProvider(provider string) (*AccountProvider, error) {
	var account AccountProvider

	err := database.Conn().
		Where("account_id = ?", a.ID).
		Where("provider = ?", provider).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, err
}
