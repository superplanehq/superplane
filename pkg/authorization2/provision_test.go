package authorization2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"

	auth "github.com/superplanehq/superplane/pkg/authorization2"
)

func TestProvision(t *testing.T) {
	database.TruncateTables()

	account, err := models.CreateAccount("Example Account", "example-account")
	require.NoError(t, err)

	org, err := models.CreateOrganization("example-org", "Example Org")
	require.NoError(t, err)

	user, err := models.CreateUser(org.ID, account.ID, "user1@example.com", "Peter Parker")
	require.NoError(t, err)

	err = auth.Provision(database.Conn(), org.ID.String(), user.ID.String())
	require.NoError(t, err)

	t.Run("creates default roles", func(t *testing.T) {
		var roles []models.RoleMetadata

		err = database.Conn().Model(&models.RoleMetadata{}).Where("domain_id = ?", org.ID.String()).Find(&roles).Error

		require.NoError(t, err)
		require.Equal(t, 3, len(roles))

		roleNames := make(map[string]bool)
		for _, role := range roles {
			roleNames[role.RoleName] = true
		}

		require.Contains(t, roleNames, "org_owner")
		require.Contains(t, roleNames, "org_admin")
		require.Contains(t, roleNames, "org_viewer")
	})

	t.Run("verify that the org owner has correct permissions", func(t *testing.T) {
		verifier, err := auth.OrgVerifier(org.ID.String(), user.ID.String())
		require.NoError(t, err)

		assertCan(t, verifier.CanCreateCanvas())
		assertCan(t, verifier.CanReadCanvas())
		assertCan(t, verifier.CanUpdateCanvas())
		assertCan(t, verifier.CanDeleteCanvas())
	})
}

func assertCan(t *testing.T, can bool, err error) {
	require.NoError(t, err)
	assert.True(t, can)
}
