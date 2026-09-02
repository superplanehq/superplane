package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/utils"
	"gorm.io/gorm"
)

var (
	ErrAccountDeleteLastUncreatedOwner    = errors.New("transfer ownership of organizations you did not create before you delete this account")
	ErrAccountDeleteLastInstallationAdmin = errors.New("promote another installation admin before you delete this account")
	ErrLastSignInMethod                   = errors.New("keep at least one sign-in method")
	ErrSignInMethodNotConnected           = errors.New("this sign-in method is not connected")
	ErrSignInIdentityInUse                = errors.New("this identity already belongs to another SuperPlane account")
	ErrAccountEmailNotFromSignInMethod    = errors.New("choose an email from a connected sign-in method")
	ErrAccountEmailInUse                  = errors.New("this email already belongs to another SuperPlane account")
)

func AccountHasPassword(tx *gorm.DB, accountID uuid.UUID) (bool, error) {
	_, err := FindAccountPasswordAuthByAccountIDInTransaction(tx, accountID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func AccountSignInMethodCount(tx *gorm.DB, account *Account) (int, error) {
	providers, err := account.GetAccountProviders(tx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, provider := range providers {
		if provider.Provider == ProviderPassword {
			continue
		}
		count++
	}
	hasPassword, err := AccountHasPassword(tx, account.ID)
	if err != nil {
		return 0, err
	}
	if hasPassword {
		count++
	}
	return count, nil
}

func (a *Account) DisconnectProvider(tx *gorm.DB, provider string) error {
	methodCount, err := AccountSignInMethodCount(tx, a)
	if err != nil {
		return err
	}
	if methodCount <= 1 {
		return ErrLastSignInMethod
	}

	var connected AccountProvider
	err = tx.Where("account_id = ?", a.ID).Where("provider = ?", provider).First(&connected).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrSignInMethodNotConnected
	}
	if err != nil {
		return err
	}

	if err := tx.Delete(&connected).Error; err != nil {
		return err
	}

	return a.reassignEmailAfterDisconnect(tx, connected.Email)
}

func (a *Account) reassignEmailAfterDisconnect(tx *gorm.DB, disconnectedEmail string) error {
	if utils.NormalizeEmail(a.Email) != utils.NormalizeEmail(disconnectedEmail) {
		return nil
	}

	remaining, err := a.GetAccountProviders(tx)
	if err != nil {
		return err
	}

	current := utils.NormalizeEmail(a.Email)
	for _, provider := range remaining {
		next := utils.NormalizeEmail(provider.Email)
		if next == "" || next == current {
			continue
		}
		err := a.SetEmail(tx, next)
		if errors.Is(err, ErrAccountEmailInUse) {
			continue
		}
		return err
	}
	return nil
}

func tombstoneEmail(accountID uuid.UUID, now time.Time) string {
	return fmt.Sprintf("deleted-%s-%d@deleted.invalid", accountID, now.Unix())
}

func (a *Account) SoftDelete(tx *gorm.DB, now time.Time) error {
	if a == nil {
		return errors.New("account is required")
	}

	createdOrgs, err := ListOrganizationsCreatedByAccount(tx, a.ID)
	if err != nil {
		return err
	}
	for _, organization := range createdOrgs {
		if err := SoftDeleteOrganizationInTransaction(tx, organization.ID.String()); err != nil {
			return err
		}
	}

	users, err := ListActiveHumanUsersForAccount(tx, a.ID)
	if err != nil {
		return err
	}
	for i := range users {
		email := tombstoneEmail(a.ID, now)
		if err := users[i].SoftDelete(tx, now, email); err != nil {
			return err
		}
	}

	if err := tx.Where("account_id = ?", a.ID).Delete(&AccountProvider{}).Error; err != nil {
		return err
	}
	if err := tx.Where("account_id = ?", a.ID).Delete(&AccountPasswordAuth{}).Error; err != nil {
		return err
	}
	if err := a.MarkPasswordChangedInTransaction(tx, now); err != nil {
		return err
	}
	if err := ClearTokenHashesForAccountInTransaction(tx, a.ID); err != nil {
		return err
	}
	if err := ClearAPIKeyTokenHashesCreatedByAccount(tx, a.ID); err != nil {
		return err
	}
	if err := DeleteUserAPITokensForAccount(tx, a.ID); err != nil {
		return err
	}

	return tx.Model(a).Updates(map[string]any{
		"email":      tombstoneEmail(a.ID, now),
		"deleted_at": now,
	}).Error
}
