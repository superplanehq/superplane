package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestAccountLinkedAccount(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("links an identity to an account", func(t *testing.T) {
		account, err := CreateAccount("Linker", "linker@example.com")
		require.NoError(t, err)

		linked := NewAccountLinkedAccount(account.ID, ProviderGitHub, "1234", "Shiroyasha", "Igor", "https://avatar")
		require.NoError(t, SaveAccountLinkedAccount(database.Conn(), linked))

		found, err := FindAccountLinkedAccount(database.Conn(), account.ID, ProviderGitHub)
		require.NoError(t, err)
		assert.Equal(t, "Shiroyasha", found.Username)
		assert.Equal(t, "shiroyasha", found.NormalizedUsername())
		assert.False(t, found.LinkedAt.IsZero())
	})

	t.Run("keeps multiple identities and activates the most recently linked identity", func(t *testing.T) {
		account, err := CreateAccount("Relinker", "relinker@example.com")
		require.NoError(t, err)

		first := NewAccountLinkedAccount(account.ID, ProviderGitHub, "1", "wrong-login", "", "")
		require.NoError(t, SaveAccountLinkedAccount(database.Conn(), first))

		second := NewAccountLinkedAccount(account.ID, ProviderGitHub, "2", "right-login", "", "")
		require.NoError(t, SaveAccountLinkedAccount(database.Conn(), second))

		linked, err := ListAccountLinkedAccounts(database.Conn(), account.ID)
		require.NoError(t, err)
		require.Len(t, linked, 2)
		assert.Equal(t, "right-login", linked[0].Username)
		assert.Equal(t, "2", linked[0].ProviderID)
		assert.True(t, linked[0].Active)
		assert.False(t, linked[1].Active)

		active, err := FindAccountLinkedAccount(database.Conn(), account.ID, ProviderGitHub)
		require.NoError(t, err)
		assert.Equal(t, "2", active.ProviderID)
	})

	t.Run("selects a previously linked identity", func(t *testing.T) {
		account, err := CreateAccount("Switcher", "switcher@example.com")
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "21", "first-login", "", ""),
		))
		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "22", "second-login", "", ""),
		))

		require.NoError(t, SelectAccountLinkedAccount(database.Conn(), account.ID, ProviderGitHub, "21"))

		active, err := FindAccountLinkedAccount(database.Conn(), account.ID, ProviderGitHub)
		require.NoError(t, err)
		assert.Equal(t, "first-login", active.Username)
	})

	t.Run("refreshes a sign-in identity without changing the selection", func(t *testing.T) {
		account, err := CreateAccount("Sign In", "sign-in@example.com")
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "31", "first-login", "", ""),
		))
		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "32", "selected-login", "", ""),
		))

		refreshed := NewAccountLinkedAccount(account.ID, ProviderGitHub, "31", "renamed-login", "", "")
		require.NoError(t, RefreshAccountLinkedAccount(database.Conn(), refreshed))

		active, err := FindAccountLinkedAccount(database.Conn(), account.ID, ProviderGitHub)
		require.NoError(t, err)
		assert.Equal(t, "32", active.ProviderID)

		linked, err := ListAccountLinkedAccounts(database.Conn(), account.ID)
		require.NoError(t, err)
		require.Len(t, linked, 2)
		assert.Equal(t, "31", linked[1].ProviderID)
		assert.Equal(t, "renamed-login", linked[1].Username)
		assert.False(t, linked[1].Active)
	})

	t.Run("allows the same identity on accounts that share no organization", func(t *testing.T) {
		owner, err := CreateAccount("Owner", "owner-shared@example.com")
		require.NoError(t, err)
		other, err := CreateAccount("Other", "other-shared@example.com")
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(owner.ID, ProviderGitHub, "9", "shared-login", "", ""),
		))

		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
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
			database.Conn(),
			NewAccountLinkedAccount(owner.ID, ProviderGitHub, "10", "taken-login", "", ""),
		))

		err = SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(other.ID, ProviderGitHub, "10", "taken-login", "", ""),
		)
		assert.ErrorIs(t, err, ErrLinkedAccountInUse)
	})

	t.Run("removes one identity and activates another identity", func(t *testing.T) {
		account, err := CreateAccount("Unlinker", "unlinker@example.com")
		require.NoError(t, err)

		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "76", "kept", "", ""),
		))
		require.NoError(t, SaveAccountLinkedAccount(
			database.Conn(),
			NewAccountLinkedAccount(account.ID, ProviderGitHub, "77", "removed", "", ""),
		))

		require.NoError(t, DeleteAccountLinkedAccount(database.Conn(), account.ID, ProviderGitHub, "77"))

		linked, err := ListAccountLinkedAccounts(database.Conn(), account.ID)
		require.NoError(t, err)
		require.Len(t, linked, 1)
		assert.Equal(t, "kept", linked[0].Username)
		assert.True(t, linked[0].Active)
	})
}
