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
	Active     bool
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
		Active:     true,
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
	err := tx.
		Where("account_id = ?", accountID).
		Order("provider ASC, active DESC, linked_at DESC").
		Find(&linked).
		Error
	if err != nil {
		return nil, err
	}
	return linked, nil
}

func FindAccountLinkedAccount(tx *gorm.DB, accountID uuid.UUID, provider string) (*AccountLinkedAccount, error) {
	var linked AccountLinkedAccount
	err := tx.
		Where("account_id = ? AND provider = ? AND active = TRUE", accountID, provider).
		First(&linked).
		Error
	if err != nil {
		return nil, err
	}
	return &linked, nil
}

// SaveAccountLinkedAccount links the identity to the account and makes it the
// active identity for its provider. Other identities for the provider remain
// linked so activity from all of them can be attributed to the account.
func SaveAccountLinkedAccount(tx *gorm.DB, linked *AccountLinkedAccount) error {
	return saveAccountLinkedAccount(tx, linked, true)
}

// RefreshAccountLinkedAccount updates an identity observed during sign-in. It
// preserves the member's explicit provider selection and activates the
// identity only when the account does not have an active identity yet.
func RefreshAccountLinkedAccount(tx *gorm.DB, linked *AccountLinkedAccount) error {
	return saveAccountLinkedAccount(tx, linked, false)
}

func SelectAccountLinkedAccount(tx *gorm.DB, accountID uuid.UUID, provider, providerID string) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		var linked AccountLinkedAccount
		err := tx.
			Where("account_id = ? AND provider = ? AND provider_id = ?", accountID, provider, providerID).
			First(&linked).
			Error
		if err != nil {
			return err
		}
		if linked.Active {
			return nil
		}
		if err := deactivateLinkedAccounts(tx, accountID, provider); err != nil {
			return err
		}
		return tx.Model(&linked).Update("active", true).Error
	})
}

func DeleteAccountLinkedAccount(tx *gorm.DB, accountID uuid.UUID, provider, providerID string) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		var linked AccountLinkedAccount
		err := tx.
			Where("account_id = ? AND provider = ? AND provider_id = ?", accountID, provider, providerID).
			First(&linked).
			Error
		if err != nil {
			return err
		}
		if err := tx.Delete(&linked).Error; err != nil {
			return err
		}
		if !linked.Active {
			return nil
		}

		var replacement AccountLinkedAccount
		err = tx.
			Where("account_id = ? AND provider = ?", accountID, provider).
			Order("linked_at DESC").
			First(&replacement).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.Model(&replacement).Update("active", true).Error
	})
}

func saveAccountLinkedAccount(tx *gorm.DB, linked *AccountLinkedAccount, activate bool) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := validateLinkedAccountOwnership(tx, linked); err != nil {
			return err
		}

		var existing AccountLinkedAccount
		err := tx.
			Where(
				"account_id = ? AND provider = ? AND provider_id = ?",
				linked.AccountID,
				linked.Provider,
				linked.ProviderID,
			).
			First(&existing).
			Error
		if err == nil {
			linked.ID = existing.ID
			if activate {
				if err := deactivateLinkedAccounts(tx, linked.AccountID, linked.Provider); err != nil {
					return err
				}
			}
			return tx.Model(&existing).Updates(map[string]any{
				"username":   linked.Username,
				"name":       linked.Name,
				"avatar_url": linked.AvatarURL,
				"active":     activate || existing.Active,
				"linked_at":  linked.LinkedAt,
			}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if activate {
			if err := deactivateLinkedAccounts(tx, linked.AccountID, linked.Provider); err != nil {
				return err
			}
			linked.Active = true
		} else {
			var activeCount int64
			if err := tx.Model(&AccountLinkedAccount{}).
				Where("account_id = ? AND provider = ? AND active = TRUE", linked.AccountID, linked.Provider).
				Count(&activeCount).
				Error; err != nil {
				return err
			}
			linked.Active = activeCount == 0
		}
		return tx.Create(linked).Error
	})
}

func validateLinkedAccountOwnership(tx *gorm.DB, linked *AccountLinkedAccount) error {
	owners := []AccountLinkedAccount{}
	err := tx.
		Where("provider = ? AND provider_id = ? AND account_id <> ?", linked.Provider, linked.ProviderID, linked.AccountID).
		Find(&owners).
		Error
	if err != nil {
		return err
	}
	for _, owner := range owners {
		conflicts, err := accountLinkedIdentityConflictsInSharedOrganization(tx, owner.AccountID, linked.AccountID)
		if err != nil {
			return err
		}
		if conflicts {
			return ErrLinkedAccountInUse
		}
	}
	return nil
}

func deactivateLinkedAccounts(tx *gorm.DB, accountID uuid.UUID, provider string) error {
	return tx.
		Model(&AccountLinkedAccount{}).
		Where("account_id = ? AND provider = ? AND active = TRUE", accountID, provider).
		Update("active", false).
		Error
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
