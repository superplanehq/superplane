package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("user not authenticated -> error", func(t *testing.T) {
		_, err := AssignRole(context.Background(), orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.User.String(), "", r.AuthService)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Equal(t, "user not authenticated", s.Message())
	})

	t.Run("user is not part of organization -> error", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, uuid.NewString(), "", r.AuthService)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("assign role with user ID", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		resp, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, newUser.ID.String(), "", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("assign role with user email", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		resp, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", newUser.GetEmail(), r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("user cannot change own role", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.User.String(), "", r.AuthService)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
		assert.Equal(t, "cannot change your own role", s.Message())
	})

	t.Run("invalid request - missing role", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, "", r.User.String(), "", r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid role", s.Message())
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", "", r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "invalid-uuid", "", r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})
}
