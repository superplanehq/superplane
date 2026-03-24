package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

type Account struct {
	ID                uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Email             string
	Name              string
	InstallationAdmin bool `gorm:"default:false"`
}

func (a *Account) IsInstallationAdmin() bool {
	return a.InstallationAdmin
}

func PromoteToInstallationAdmin(accountID string) error {
	return database.Conn().
		Model(&Account{}).
		Where("id = ?", accountID).
		Update("installation_admin", true).
		Error
}

func DemoteFromInstallationAdmin(accountID string) error {
	return database.Conn().
		Model(&Account{}).
		Where("id = ?", accountID).
		Update("installation_admin", false).
		Error
}

func CreateAccount(name, email string) (*Account, error) {
	return CreateAccountInTransaction(database.Conn(), name, email)
}

func CreateAccountInTransaction(tx *gorm.DB, name, email string) (*Account, error) {
	account := &Account{Name: name, Email: utils.NormalizeEmail(email)}
	err := tx.Create(account).Error
	if err != nil {
		return nil, err
	}

	return account, nil
}

func ListAccounts(search string, limit, offset int) ([]Account, int64, error) {
	query := database.Conn().Model(&Account{})

	if search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	var accounts []Account
	if err := query.Order("name ASC").Find(&accounts).Error; err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
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
		Where("email = ?", utils.NormalizeEmail(email)).
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
		Where("email = ?", a.Email).
		Where("state = ?", InvitationStatePending).
		Find(&invitations).
		Error

	if err != nil {
		return nil, err
	}

	return invitations, nil
}

func FindAccountByProvider(provider, providerID string) (*Account, error) {
	var accountProvider AccountProvider
	err := database.Conn().
		Where("provider = ?", provider).
		Where("provider_id = ?", providerID).
		First(&accountProvider).
		Error

	if err != nil {
		return nil, err
	}

	return FindAccountByID(accountProvider.AccountID.String())
}

func (a *Account) UpdateEmail(newEmail string) error {
	normalizedEmail := utils.NormalizeEmail(newEmail)
	originalEmail := a.Email

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		err := tx.Model(a).Update("email", normalizedEmail).Error
		if err != nil {
			return err
		}

		err = tx.Model(&User{}).
			Where("account_id = ?", a.ID).
			Update("email", normalizedEmail).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err == nil {
		a.Email = normalizedEmail
		return nil
	}

	a.Email = originalEmail
	return err
}

func (a *Account) UpdateEmailForProvider(newEmail, provider, providerID string) error {
	normalizedEmail := utils.NormalizeEmail(newEmail)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {

		err := tx.Model(a).Update("email", normalizedEmail).Error
		if err != nil {
			return err
		}

		err = tx.Model(&User{}).
			Where("account_id = ?", a.ID).
			Update("email", normalizedEmail).Error
		if err != nil {
			return err
		}

		err = tx.Model(&AccountProvider{}).
			Where("account_id = ? AND provider = ? AND provider_id = ?", a.ID, provider, providerID).
			Update("email", normalizedEmail).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err == nil {
		a.Email = normalizedEmail
	}

	return err
}
