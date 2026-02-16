package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/utils"
)

func TestFindAccountByProvider(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("should find account by provider info", func(t *testing.T) {

		account, err := CreateAccount("Test User", "test@example.com")
		require.NoError(t, err)

		provider := &AccountProvider{
			AccountID:    account.ID,
			Provider:     "github",
			ProviderID:   "12345",
			Username:     "testuser",
			Email:        account.Email,
			Name:         account.Name,
			AvatarURL:    "https://github.com/testuser.png",
			AccessToken:  "token123",
			RefreshToken: "refresh123",
		}
		err = database.Conn().Create(provider).Error
		require.NoError(t, err)

		foundAccount, err := FindAccountByProvider("github", "12345")
		require.NoError(t, err)
		assert.Equal(t, account.ID, foundAccount.ID)
		assert.Equal(t, account.Email, foundAccount.Email)
		assert.Equal(t, account.Name, foundAccount.Name)
	})

	t.Run("should return error when provider not found", func(t *testing.T) {
		account, err := FindAccountByProvider("nonexistent", "99999")
		assert.Error(t, err)
		assert.Nil(t, account)
	})

	t.Run("should return error when account is deleted", func(t *testing.T) {

		deletedAccount, err := CreateAccount("Deleted User", "deleted@example.com")
		require.NoError(t, err)

		provider := &AccountProvider{
			AccountID:   deletedAccount.ID,
			Provider:    "google",
			ProviderID:  "67890",
			Username:    "deleteduser",
			Email:       deletedAccount.Email,
			Name:        deletedAccount.Name,
			AccessToken: "token456",
		}
		err = database.Conn().Create(provider).Error
		require.NoError(t, err)

		err = database.Conn().Delete(provider).Error
		require.NoError(t, err)

		err = database.Conn().Delete(deletedAccount).Error
		require.NoError(t, err)

		account, err := FindAccountByProvider("google", "67890")
		assert.Error(t, err)
		assert.Nil(t, account)
	})
}

func TestAccount_UpdateEmail(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("should update account and all related user emails", func(t *testing.T) {

		orgID := uuid.New()
		organization := &Organization{
			ID:   orgID,
			Name: "Test Org",
		}
		err := database.Conn().Create(organization).Error
		require.NoError(t, err)

		account, err := CreateAccount("Test User", "original@example.com")
		require.NoError(t, err)

		user := &User{
			OrganizationID: orgID,
			AccountID:      &account.ID,
			Email:          &account.Email,
			Name:           account.Name,
		}
		err = database.Conn().Create(user).Error
		require.NoError(t, err)

		otherAccount, err := CreateAccount("Other User", "other@example.com")
		require.NoError(t, err)
		otherUser := &User{
			OrganizationID: orgID,
			AccountID:      &otherAccount.ID,
			Email:          &otherAccount.Email,
			Name:           otherAccount.Name,
		}
		err = database.Conn().Create(otherUser).Error
		require.NoError(t, err)

		newEmail := "newemail@example.com"
		normalizedNewEmail := utils.NormalizeEmail(newEmail)

		assert.NotEqual(t, normalizedNewEmail, account.Email)

		err = account.UpdateEmail(newEmail)
		require.NoError(t, err)

		assert.Equal(t, normalizedNewEmail, account.Email)

		var accountFromDB Account
		err = database.Conn().Where("id = ?", account.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, normalizedNewEmail, accountFromDB.Email)

		var userFromDB User
		err = database.Conn().Where("id = ?", user.ID).First(&userFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, normalizedNewEmail, userFromDB.GetEmail())

		var otherUserFromDB User
		err = database.Conn().Where("id = ?", otherUser.ID).First(&otherUserFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, otherAccount.Email, otherUserFromDB.GetEmail())
		assert.NotEqual(t, normalizedNewEmail, otherUserFromDB.GetEmail())
	})

	t.Run("should normalize email", func(t *testing.T) {
		account, err := CreateAccount("Test User", "test@example.com")
		require.NoError(t, err)

		err = account.UpdateEmail("UPPERCASE@EXAMPLE.COM")
		require.NoError(t, err)

		assert.Equal(t, "uppercase@example.com", account.Email)

		var accountFromDB Account
		err = database.Conn().Where("id = ?", account.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, "uppercase@example.com", accountFromDB.Email)
	})

	t.Run("should handle transaction rollback on failure", func(t *testing.T) {
		account, err := CreateAccount("Test User", "test2@example.com")
		require.NoError(t, err)
		originalEmail := account.Email

		conflictEmail := "conflict@example.com"
		_, err = CreateAccount("Conflict User", conflictEmail)
		require.NoError(t, err)

		err = account.UpdateEmail(conflictEmail)
		require.Error(t, err, "Expected error due to unique constraint violation")

		assert.Equal(t, originalEmail, account.Email)

		var accountFromDB Account
		err = database.Conn().Where("id = ?", account.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, originalEmail, accountFromDB.Email)
	})
}

func TestAccount_UpdateEmailForProvider(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("should update account, users, and only specific provider", func(t *testing.T) {

		orgID := uuid.New()
		organization := &Organization{
			ID:   orgID,
			Name: "Test Org",
		}
		err := database.Conn().Create(organization).Error
		require.NoError(t, err)

		account, err := CreateAccount("Test User", "original@example.com")
		require.NoError(t, err)

		user := &User{
			OrganizationID: orgID,
			AccountID:      &account.ID,
			Email:          &account.Email,
			Name:           account.Name,
		}
		err = database.Conn().Create(user).Error
		require.NoError(t, err)

		githubProvider := &AccountProvider{
			AccountID:  account.ID,
			Provider:   "github",
			ProviderID: "github123",
			Username:   "testuser",
			Email:      account.Email,
			Name:       account.Name,
		}
		err = database.Conn().Create(githubProvider).Error
		require.NoError(t, err)

		googleProvider := &AccountProvider{
			AccountID:  account.ID,
			Provider:   "google",
			ProviderID: "google456",
			Username:   "testuser2",
			Email:      account.Email,
			Name:       account.Name,
		}
		err = database.Conn().Create(googleProvider).Error
		require.NoError(t, err)

		originalEmail := account.Email

		newEmail := "newemail@example.com"
		normalizedNewEmail := utils.NormalizeEmail(newEmail)

		err = account.UpdateEmailForProvider(newEmail, "github", "github123")
		require.NoError(t, err)

		assert.Equal(t, normalizedNewEmail, account.Email)

		var accountFromDB Account
		err = database.Conn().Where("id = ?", account.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, normalizedNewEmail, accountFromDB.Email)

		var userFromDB User
		err = database.Conn().Where("id = ?", user.ID).First(&userFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, normalizedNewEmail, userFromDB.GetEmail())

		var githubProviderFromDB AccountProvider
		err = database.Conn().Where("id = ?", githubProvider.ID).First(&githubProviderFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, normalizedNewEmail, githubProviderFromDB.Email)

		var googleProviderFromDB AccountProvider
		err = database.Conn().Where("id = ?", googleProvider.ID).First(&googleProviderFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, originalEmail, googleProviderFromDB.Email, "Google provider should keep original email")
	})
}
