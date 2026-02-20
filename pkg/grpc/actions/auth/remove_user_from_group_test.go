package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_RemoveUserFromGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	// Create a group first
	newUser := support.CreateUser(t, r, r.Organization.ID)
	groupName := support.RandomName("group")
	require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgAdmin, "", ""))

	t.Run("user is not part of group -> error", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, newUser.ID.String(), "", groupName, r.AuthService)
		require.Error(t, err)
	})

	t.Run("remove user from group with user ID", func(t *testing.T) {
		require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, newUser.ID.String(), groupName))
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, newUser.ID.String(), "", groupName, r.AuthService)
		require.NoError(t, err)
	})

	t.Run("remove user from group with user email", func(t *testing.T) {
		require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, newUser.ID.String(), groupName))
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", newUser.GetEmail(), groupName, r.AuthService)
		require.NoError(t, err)
	})

	t.Run("user not found by email", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", "nonexistent-user@test.com", groupName, r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, r.User.String(), "", "", r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "group name must be specified", s.Message())
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", "", groupName, r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveUserFromGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "invalid-uuid", "", groupName, r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})
}
