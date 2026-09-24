package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrLinkedAccountInUse reports that the external identity is already linked to
// another member in a shared organization. Velocity credits one GitHub login to
// one member per organization.
var ErrLinkedAccountInUse = errors.New("linked account belongs to another account")

// AccountLinkedAccount is an identity a member owns on another service. It is
// not a sign-in method: it grants no session and stores no token. SuperPlane
// uses it to attribute activity, such as repository authorship in Velocity.
type AccountLinkedAccount struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AccountID  uuid.UUID
	Provider   string
	ProviderID string
	Username   string
	Name       string
	AvatarURL  string
	LinkedAt   time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewAccountLinkedAccount(accountID uuid.UUID, provider, providerID, username, name, avatarURL string) *AccountLinkedAccount {
	return &AccountLinkedAccount{
		AccountID:  accountID,
		Provider:   provider,
		ProviderID: providerID,
		Username:   username,
		Name:       name,
		AvatarURL:  avatarURL,
		LinkedAt:   time.Now(),
	}
}

func (a *AccountLinkedAccount) TableName() string {
	return "account_linked_accounts"
}

// NormalizedUsername matches how the Velocity report resolves a login.
func (a *AccountLinkedAccount) NormalizedUsername() string {
	return strings.ToLower(strings.TrimSpace(a.Username))
}

func ListAccountLinkedAccounts(tx *gorm.DB, accountID uuid.UUID) ([]AccountLinkedAccount, error) {
	linked := []AccountLinkedAccount{}
	err := tx.Where("account_id = ?", accountID).Order("provider ASC").Find(&linked).Error
	if err != nil {
		return nil, err
	}
	return linked, nil
}

func FindAccountLinkedAccount(tx *gorm.DB, accountID uuid.UUID, provider string) (*AccountLinkedAccount, error) {
	var linked AccountLinkedAccount
	err := tx.Where("account_id = ? AND provider = ?", accountID, provider).First(&linked).Error
	if err != nil {
		return nil, err
	}
	return &linked, nil
}

func accountLinkedIdentityConflictsInSharedOrganization(tx *gorm.DB, ownerAccountID, claimantAccountID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM users owner_user
			INNER JOIN users claimant_user ON owner_user.organization_id = claimant_user.organization_id
			WHERE owner_user.account_id = ?
				AND claimant_user.account_id = ?
				AND owner_user.type = ?
				AND claimant_user.type = ?
				AND owner_user.deleted_at IS NULL
				AND claimant_user.deleted_at IS NULL
		)
	`, ownerAccountID, claimantAccountID, UserTypeHuman, UserTypeHuman).Scan(&exists).Error
	return exists, err
}

// SaveAccountLinkedAccount links the identity to the account. It replaces the
// identity the account previously linked for the same provider, so a member can
// correct a wrong link without an extra step.
func SaveAccountLinkedAccount(tx *gorm.DB, linked *AccountLinkedAccount) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		var owner AccountLinkedAccount
		err := tx.
			Where("provider = ? AND provider_id = ?", linked.Provider, linked.ProviderID).
			First(&owner).
			Error
		if err == nil && owner.AccountID != linked.AccountID {
			conflicts, conflictErr := accountLinkedIdentityConflictsInSharedOrganization(tx, owner.AccountID, linked.AccountID)
			if conflictErr != nil {
				return conflictErr
			}
			if conflicts {
				return ErrLinkedAccountInUse
			}
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var existing AccountLinkedAccount
		err = tx.
			Where("account_id = ? AND provider = ?", linked.AccountID, linked.Provider).
			First(&existing).
			Error
		if err == nil {
			linked.ID = existing.ID
			return tx.Model(&existing).Updates(map[string]any{
				"provider_id": linked.ProviderID,
				"username":    linked.Username,
				"name":        linked.Name,
				"avatar_url":  linked.AvatarURL,
				"linked_at":   linked.LinkedAt,
			}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return tx.Create(linked).Error
	})
}

func DeleteAccountLinkedAccount(tx *gorm.DB, accountID uuid.UUID, provider string) error {
	return tx.
		Where("account_id = ? AND provider = ?", accountID, provider).
		Delete(&AccountLinkedAccount{}).
		Error
}
