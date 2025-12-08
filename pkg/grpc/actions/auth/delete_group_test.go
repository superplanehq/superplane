package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_DeleteOrganizationGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("successful group deletion", func(t *testing.T) {
		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test Group")
		require.NoError(t, err)

		userID := uuid.New().String()
		err = r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID, "test-group")
		require.NoError(t, err)

		groups, err := r.AuthService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")

		users, err := r.AuthService.GetGroupUsers(orgID, models.DomainTypeOrganization, "test-group")
		require.NoError(t, err)
		assert.Contains(t, users, userID)

		resp, err := DeleteGroup(ctx, models.DomainTypeOrganization, orgID, "test-group", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		users, err = r.AuthService.GetGroupUsers(orgID, models.DomainTypeOrganization, "test-group")
		require.NoError(t, err)
		assert.Empty(t, users)

		// Verify the group no longer exists in the groups list
		groups, err = r.AuthService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.NotContains(t, groups, "test-group")
	})

	t.Run("delete non-existent group", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeOrganization, orgID, "non-existent-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := DeleteGroup(ctx, models.DomainTypeOrganization, orgID, "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - invalid organization ID for group", func(t *testing.T) {
		err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test Group")
		require.NoError(t, err)

		_, err = DeleteGroup(ctx, models.DomainTypeOrganization, "invalid-uuid", "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})
}
