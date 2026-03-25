package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestInstallationAdmin(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("new accounts are not installation admins by default", func(t *testing.T) {
		account, err := CreateAccount("Regular User", "regular@example.com")
		require.NoError(t, err)
		assert.False(t, account.IsInstallationAdmin())
	})

	t.Run("PromoteToInstallationAdmin sets the flag", func(t *testing.T) {
		account, err := CreateAccount("Admin Candidate", "candidate@example.com")
		require.NoError(t, err)
		assert.False(t, account.IsInstallationAdmin())

		err = PromoteToInstallationAdmin(account.ID.String())
		require.NoError(t, err)

		// Re-fetch from database
		refreshed, err := FindAccountByID(account.ID.String())
		require.NoError(t, err)
		assert.True(t, refreshed.IsInstallationAdmin())
	})

	t.Run("DemoteFromInstallationAdmin clears the flag", func(t *testing.T) {
		account, err := CreateAccount("Temp Admin", "temp-admin@example.com")
		require.NoError(t, err)

		err = PromoteToInstallationAdmin(account.ID.String())
		require.NoError(t, err)

		err = DemoteFromInstallationAdmin(account.ID.String())
		require.NoError(t, err)

		refreshed, err := FindAccountByID(account.ID.String())
		require.NoError(t, err)
		assert.False(t, refreshed.IsInstallationAdmin())
	})

	t.Run("IsInstallationAdmin returns correct value", func(t *testing.T) {
		nonAdmin := &Account{InstallationAdmin: false}
		assert.False(t, nonAdmin.IsInstallationAdmin())

		admin := &Account{InstallationAdmin: true}
		assert.True(t, admin.IsInstallationAdmin())
	})
}

func TestListAllOrganizations(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("returns empty list when no organizations exist", func(t *testing.T) {
		orgs, total, err := ListAllOrganizations("", 50, 0)
		require.NoError(t, err)
		assert.Empty(t, orgs)
		assert.Equal(t, int64(0), total)
	})

	t.Run("returns all organizations ordered by name", func(t *testing.T) {
		_, err := CreateOrganization("Zebra Org", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Alpha Org", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Middle Org", "")
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations("", 50, 0)
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Alpha Org", orgs[0].Name)
		assert.Equal(t, "Middle Org", orgs[1].Name)
		assert.Equal(t, "Zebra Org", orgs[2].Name)
	})

	t.Run("excludes soft-deleted organizations", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		org, err := CreateOrganization("Active Org", "")
		require.NoError(t, err)
		toDelete, err := CreateOrganization("Deleted Org", "")
		require.NoError(t, err)

		err = SoftDeleteOrganization(toDelete.ID.String())
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations("", 50, 0)
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, org.ID, orgs[0].ID)
	})

	t.Run("filters by search term", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		_, err := CreateOrganization("Alpha Corp", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Beta Inc", "")
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations("alpha", 50, 0)
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Alpha Corp", orgs[0].Name)
	})

	t.Run("paginates results", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		_, err := CreateOrganization("Aaa", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Bbb", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Ccc", "")
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations("", 2, 0)
		require.NoError(t, err)
		assert.Len(t, orgs, 2)
		assert.Equal(t, int64(3), total)

		orgs2, _, err := ListAllOrganizations("", 2, 2)
		require.NoError(t, err)
		assert.Len(t, orgs2, 1)
	})
}

func TestListActiveUsersByOrganization(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("returns human users for organization", func(t *testing.T) {
		org, err := CreateOrganization("Test Org", "")
		require.NoError(t, err)

		account, err := CreateAccount("Test User", "user@example.com")
		require.NoError(t, err)

		_, err = CreateUser(org.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)

		users, total, err := ListActiveUsersByOrganization(org.ID.String(), "", 50, 0)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Test User", users[0].Name)
	})

	t.Run("excludes service accounts", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		org, err := CreateOrganization("SA Test Org", "")
		require.NoError(t, err)

		account, err := CreateAccount("Human", "human@example.com")
		require.NoError(t, err)

		_, err = CreateUser(org.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)

		saEmail := "sa@example.com"
		sa := &User{
			OrganizationID: org.ID,
			Email:          &saEmail,
			Name:           "Bot",
			Type:           UserTypeServiceAccount,
		}
		err = database.Conn().Create(sa).Error
		require.NoError(t, err)

		users, total, err := ListActiveUsersByOrganization(org.ID.String(), "", 50, 0)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Human", users[0].Name)
	})
}
