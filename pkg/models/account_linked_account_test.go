package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestAccountLinkedAccount(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())

	t.Run("links an identity to an account", func(t *testing.T) {
		account, err := CreateAccount("Linker", "linker@example.com")
		require.NoError(t, err)

		linked := NewAccountLinkedAccount(account.ID, ProviderGitHub, "1234", "Shiroyasha", "Igor", "https://avatar")
		require.NoError(t, SaveAccountLinkedAccount(db, linked))

		found, err := FindAccountLinkedAccount(db, account.ID, ProviderGitHub)
		require.NoError(t, err)
		assert.Equal(t, "Shiroyasha", found.Username)
		assert.Equal(t, "shiroyasha", found.NormalizedUsername())
		assert.False(t, found.LinkedAt.IsZero())
	})

	t.Run("replaces the identity the account linked before", func(t *testing.T) {
		account, err := CreateAccount("Relinker", "relinker@example.com")
		require.NoError(t, err)

		first := NewAccountLinkedAccount(account.ID, ProviderGitHub, "1", "wrong-login", "", "")
		require.NoError(t, SaveAccountLinkedAccount(db, first))

		second := NewAccountLinkedAccount(account.ID, ProviderGitHub, "2", "right-login", "", "")
		require.NoError(t, SaveAccountLinkedAccount(db, second))

		linked, err := ListAccountLinkedAccounts(db, account.ID)
		require.NoError(t, err)
		require.Len(t, linked, 1)
		assert.Equal(t, "right-login", linked[0].Username)
		assert.Equal(t, "2", linked[0].ProviderID)
	})

	t.Run("allows the same identity on accounts that share no organization", func(t *testing.T) {
		owner, err := CreateAccount("Owner", "owner-shared@example.com")
		require.NoError(t, err)
		other, err := CreateAccount("Other", "other-shared@example.com")
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			db,
			NewAccountLinkedAccount(owner.ID, ProviderGitHub, "9", "shared-login", "", ""),
		))

		require.NoError(t, SaveAccountLinkedAccount(
			db,
			NewAccountLinkedAccount(other.ID, ProviderGitHub, "9", "shared-login", "", ""),
		))
	})

	t.Run("refuses an identity when both accounts belong to the same organization", func(t *testing.T) {
		owner, err := CreateAccount("Owner", "owner-org@example.com")
		require.NoError(t, err)
		other, err := CreateAccount("Other", "other-org@example.com")
		require.NoError(t, err)

		org, err := CreateOrganization("Shared Org", "")
		require.NoError(t, err)
		_, err = CreateUser(org.ID, owner.ID, owner.Email, owner.Name)
		require.NoError(t, err)
		_, err = CreateUser(org.ID, other.ID, other.Email, other.Name)
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			db,
			NewAccountLinkedAccount(owner.ID, ProviderGitHub, "10", "taken-login", "", ""),
		))

		err = SaveAccountLinkedAccount(
			db,
			NewAccountLinkedAccount(other.ID, ProviderGitHub, "10", "taken-login", "", ""),
		)
		assert.ErrorIs(t, err, ErrLinkedAccountInUse)
	})

	t.Run("removes the link", func(t *testing.T) {
		account, err := CreateAccount("Unlinker", "unlinker@example.com")
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			db,
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "77", "gone", "", ""),
		))

		require.NoError(t, DeleteAccountLinkedAccount(db, account.ID, ProviderGitHub))

		linked, err := ListAccountLinkedAccounts(db, account.ID)
		require.NoError(t, err)
		assert.Empty(t, linked)
	})

	t.Run("prefers a linked GitHub identity", func(t *testing.T) {
		account, err := CreateAccount("Identity", "identity@example.com")
		require.NoError(t, err)
		require.NoError(t, db.Create(&AccountProvider{
			AccountID:  account.ID,
			Provider:   ProviderGitHub,
			ProviderID: "10",
			Username:   "sign-in-login",
		}).Error)
		require.NoError(t, SaveAccountLinkedAccount(
			db,
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "20", "linked-login", "", ""),
		))

		identity, err := FindAccountGitHubIdentity(db, account.ID)
		require.NoError(t, err)
		assert.Equal(t, "20", identity.ProviderID)
		assert.Equal(t, "linked-login", identity.Username)
	})

	t.Run("falls back to the GitHub sign-in identity", func(t *testing.T) {
		account, err := CreateAccount("Signed in", "signed-in@example.com")
		require.NoError(t, err)
		require.NoError(t, db.Create(&AccountProvider{
			AccountID:  account.ID,
			Provider:   ProviderGitHub,
			ProviderID: "30",
			Username:   "provider-login",
		}).Error)

		identity, err := FindAccountGitHubIdentity(db, account.ID)
		require.NoError(t, err)
		assert.Equal(t, "30", identity.ProviderID)
		assert.Equal(t, "provider-login", identity.Username)
	})
}
