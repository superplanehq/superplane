package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_RemoveRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("user is not part of organization -> error", func(t *testing.T) {
		_, err := RemoveRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, uuid.NewString(), "", r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("remove role with user ID", func(t *testing.T) {
		user := support.CreateUser(t, r.Organization.ID)
		require.NoError(t, r.AuthService.AssignRole(user.ID.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization))
		resp, err := RemoveRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, user.ID.String(), "", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("remove role with user email", func(t *testing.T) {
		user := support.CreateUser(t, r.Organization.ID)
		require.NoError(t, r.AuthService.AssignRole(user.ID.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization))
		resp, err := RemoveRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", user.Email, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := RemoveRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", "", r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := RemoveRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "invalid-uuid", "", r.AuthService)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})
}
