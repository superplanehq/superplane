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
		logs := support.CaptureSecurityLogs(t)
		newUser := support.CreateUser(t, r, r.Organization.ID)
		resp, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, "", newUser.GetEmail(), r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		events := support.SecurityEvents(t, logs)
		require.Len(t, events, 1)
		assert.Equal(t, "role.assign", events[0]["action"])
		assert.Equal(t, "success", events[0]["outcome"])
		assert.Equal(t, r.User.String(), events[0]["actor_user_id"])
		assert.Equal(t, newUser.ID.String(), events[0]["target_id"])
		assert.Equal(t, orgID, events[0]["domain_id"])
		assert.Equal(t, models.RoleOrgAdmin, events[0]["role_name"])
	})

	t.Run("user cannot change own role", func(t *testing.T) {
		logs := support.CaptureSecurityLogs(t)
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.User.String(), "", r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, code)
		assert.Equal(t, "cannot change your own role", msg)
		assert.Empty(t, support.SecurityEvents(t, logs))
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
}
