package actions

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_CheckPermission(t *testing.T) {
	r := support.Setup(t)
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role to user
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgAdmin, orgID, authorization.DomainOrg)
	require.NoError(t, err)

	t.Run("permission allowed", func(t *testing.T) {
		req := &pb.CheckPermissionRequest{
			UserId:     r.User.String(),
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Resource:   "canvas",
			Action:     "read",
		}

		resp, err := CheckPermission(ctx, req, authService)
		require.NoError(t, err)
		assert.True(t, resp.Allowed)
	})

	t.Run("permission denied", func(t *testing.T) {
		req := &pb.CheckPermissionRequest{
			UserId:     r.User.String(),
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Resource:   "member",
			Action:     "remove",
		}

		resp, err := CheckPermission(ctx, req, authService)
		require.NoError(t, err)
		assert.False(t, resp.Allowed)
	})

	t.Run("invalid request - missing resource", func(t *testing.T) {
		req := &pb.CheckPermissionRequest{
			UserId:     r.User.String(),
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Resource:   "",
			Action:     "read",
		}

		_, err := CheckPermission(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource and action must be specified")
	})
}
