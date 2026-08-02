package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("user not authenticated -> error", func(t *testing.T) {
		_, err := AssignRole(context.Background(), orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.User.String(), "", r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
		assert.Equal(t, "user not authenticated", msg)
	})

	t.Run("user is not part of organization -> error", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, uuid.NewString(), "", r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "user not found", msg)
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
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot change your own role", msg)
	})

	t.Run("invalid request - missing role", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, "", r.User.String(), "", r.AuthService)
		assert.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "invalid role", msg)
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", "", r.AuthService)
		assert.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "user not found", msg)
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "invalid-uuid", "", r.AuthService)
		assert.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "user not found", msg)
	})

	t.Run("admin cannot assign org_owner", func(t *testing.T) {
		admin := support.CreateUser(t, r, r.Organization.ID)
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, admin.ID.String(), "", r.AuthService)
		require.NoError(t, err)

		target := support.CreateUser(t, r, r.Organization.ID)
		adminCtx := authentication.SetUserIdInMetadata(context.Background(), admin.ID.String())
		_, err = AssignRole(adminCtx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgOwner, target.ID.String(), "", r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot assign a role with permissions you do not have", msg)
	})

	t.Run("admin can assign org_admin", func(t *testing.T) {
		admin := support.CreateUser(t, r, r.Organization.ID)
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, admin.ID.String(), "", r.AuthService)
		require.NoError(t, err)

		target := support.CreateUser(t, r, r.Organization.ID)
		adminCtx := authentication.SetUserIdInMetadata(context.Background(), admin.ID.String())
		resp, err := AssignRole(adminCtx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, target.ID.String(), "", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("cannot demote the last organization owner", func(t *testing.T) {
		admin := support.CreateUser(t, r, r.Organization.ID)
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, admin.ID.String(), "", r.AuthService)
		require.NoError(t, err)

		adminCtx := authentication.SetUserIdInMetadata(context.Background(), admin.ID.String())
		_, err = AssignRole(adminCtx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgViewer, r.User.String(), "", r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "cannot demote the last organization owner", msg)

		secondOwner := support.CreateUser(t, r, r.Organization.ID)
		_, err = AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgOwner, secondOwner.ID.String(), "", r.AuthService)
		require.NoError(t, err)

		_, err = AssignRole(adminCtx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, secondOwner.ID.String(), "", r.AuthService)
		require.NoError(t, err)

		_, err = AssignRole(adminCtx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgViewer, r.User.String(), "", r.AuthService)
		code, msg, ok = grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "cannot demote the last organization owner", msg)
	})
}
