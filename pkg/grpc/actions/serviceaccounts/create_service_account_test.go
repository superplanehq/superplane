package serviceaccounts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// authedContext builds an incoming-metadata context carrying both the user and
// organization identifiers that CreateServiceAccount expects.
func authedContext(userID, orgID string) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs("x-user-id", userID, "x-organization-id", orgID),
	)
}

func Test__CreateServiceAccount(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	ctx := authedContext(r.User.String(), orgID)

	t.Run("unauthenticated user", func(t *testing.T) {
		_, err := CreateServiceAccount(context.Background(), &pb.CreateServiceAccountRequest{
			Name: support.RandomName("sa"),
			Role: models.RoleOrgAdmin,
		}, r.AuthService)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})

	t.Run("role is required", func(t *testing.T) {
		_, err := CreateServiceAccount(ctx, &pb.CreateServiceAccountRequest{
			Name: support.RandomName("sa"),
		}, r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "role is required", msg)
	})

	t.Run("rejects a role that does not exist", func(t *testing.T) {
		_, err := CreateServiceAccount(ctx, &pb.CreateServiceAccountRequest{
			Name: support.RandomName("sa"),
			Role: "does_not_exist",
		}, r.AuthService)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "invalid role for service account", msg)
	})

	t.Run("creates a service account with a default role", func(t *testing.T) {
		resp, err := CreateServiceAccount(ctx, &pb.CreateServiceAccountRequest{
			Name: support.RandomName("admin-bot"),
			Role: models.RoleOrgAdmin,
		}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp.ServiceAccount)
		assert.NotEmpty(t, resp.Token)
	})

	// Regression test for #4674: custom roles must be assignable to service
	// accounts, not just the hardcoded org_admin/org_viewer defaults.
	t.Run("creates a service account with a custom role", func(t *testing.T) {
		roleName := support.RandomName("custom-role")
		require.NoError(t, r.AuthService.CreateCustomRole(orgID, &authorization.RoleDefinition{
			Name:        roleName,
			DisplayName: "Release Bot",
			DomainType:  models.DomainTypeOrganization,
			Description: "custom role for tests",
			Permissions: []*authorization.Permission{
				{Resource: "canvases", Action: "read", DomainType: models.DomainTypeOrganization},
			},
		}))

		resp, err := CreateServiceAccount(ctx, &pb.CreateServiceAccountRequest{
			Name: support.RandomName("release-bot"),
			Role: roleName,
		}, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp.ServiceAccount)

		allowed, err := r.AuthService.CheckOrganizationPermission(ctx, resp.ServiceAccount.Id, orgID, "canvases", "read")
		require.NoError(t, err)
		assert.True(t, allowed, "the created service account should inherit the custom role's permissions")
	})
}
