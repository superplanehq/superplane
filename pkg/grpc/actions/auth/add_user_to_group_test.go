package auth

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/protos/users"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_AddUserToGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	groupName := support.RandomName("test-group")
	err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, groupName, models.RoleOrgAdmin, "", "")
	require.NoError(t, err)

	t.Run("add user to group with user ID", func(t *testing.T) {
		newUser := support.CreateUser(t, r.Organization.ID)
		_, err := AddUserToGroup(ctx, orgID, models.DomainTypeOrganization, orgID, newUser.ID.String(), "", groupName, r.AuthService)
		require.NoError(t, err)

		response, err := ListGroupUsers(context.Background(), models.DomainTypeOrganization, orgID, groupName, r.AuthService)
		require.NoError(t, err)
		assert.True(t, slices.ContainsFunc(response.Users, func(user *users.User) bool {
			return user.Metadata.Id == newUser.ID.String() && user.Metadata.Email == newUser.Email
		}))
	})

	t.Run("add user to organization group with email", func(t *testing.T) {
		newUser := support.CreateUser(t, r.Organization.ID)
		_, err := AddUserToGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", newUser.Email, groupName, r.AuthService)
		require.NoError(t, err)

		response, err := ListGroupUsers(context.Background(), models.DomainTypeOrganization, orgID, groupName, r.AuthService)
		require.NoError(t, err)
		assert.True(t, slices.ContainsFunc(response.Users, func(user *users.User) bool {
			return user.Metadata.Id == newUser.ID.String() && user.Metadata.Email == newUser.Email
		}))
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, orgID, models.DomainTypeOrganization, orgID, r.User.String(), "", "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "", "", groupName, r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AddUserToGroup(ctx, orgID, models.DomainTypeOrganization, orgID, "invalid-uuid", "", groupName, r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("canvas group does not exist -> error", func(t *testing.T) {
		newUser := support.CreateUser(t, r.Organization.ID)
		_, err = AddUserToGroup(ctx, orgID, models.DomainTypeCanvas, r.Canvas.ID.String(), newUser.ID.String(), "", "non-existent-group", r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unknown, s.Code())
		assert.Contains(t, s.Message(), "group non-existent-group does not exist")
	})

	t.Run("add user to canvas group", func(t *testing.T) {
		groupName := support.RandomName("canvas-group")
		err = r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, groupName, models.RoleCanvasAdmin, "", "")
		require.NoError(t, err)

		newUser := support.CreateUser(t, r.Organization.ID)
		_, err = AddUserToGroup(ctx, orgID, models.DomainTypeCanvas, r.Canvas.ID.String(), newUser.ID.String(), "", groupName, r.AuthService)
		require.NoError(t, err)

		response, err := ListGroupUsers(context.Background(), models.DomainTypeCanvas, r.Canvas.ID.String(), groupName, r.AuthService)
		require.NoError(t, err)
		assert.Len(t, response.Users, 1)
		assert.True(t, slices.ContainsFunc(response.Users, func(user *users.User) bool {
			return user.Metadata.Id == newUser.ID.String() && user.Metadata.Email == newUser.Email
		}))
	})
}
