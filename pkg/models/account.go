package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type Account struct {
	ID    uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Email string
	Name  string
}

func CreateAccount(name, email string) (*Account, error) {
	account := &Account{Name: name, Email: email}
	err := database.Conn().Create(account).Error
	if err != nil {
		return nil, err
	}

	return account, nil
}

func FindAccountByID(id string) (*Account, error) {
	var account Account

	err := database.Conn().
		Where("id = ?", id).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func FindAccountByEmail(email string) (*Account, error) {
	var account Account

	err := database.Conn().
		Where("email = ?", email).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (a *Account) GetAccountProviders() ([]AccountProvider, error) {
	providers := []AccountProvider{}

	err := database.Conn().
		Where("account_id = ?", a.ID).
		Find(&providers).
		Error

	if err != nil {
		return nil, err
	}

	return providers, nil
}

func (a *Account) GetAccountProvider(provider string) (*AccountProvider, error) {
	var account AccountProvider
	err := database.Conn().
		Where("account_id = ?", a.ID, provider).
		Where("provider = ?", provider).
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
		Where("account_id = ?", a.ID).
		Where("provider = ?", provider).
		Where("provider_id = ?", providerID).
		First(&account).
		Error

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (a *Account) FindPendingInvitations() ([]OrganizationInvitation, error) {
	invitations := []OrganizationInvitation{}

	err := database.Conn().
		Where("account_id = ?", a.ID).
		Where("status = ?", InvitationStatusPending).
		Find(&invitations).
		Error

	if err != nil {
		return nil, err
	}

	return invitations, nil
}
