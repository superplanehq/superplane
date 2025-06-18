package auth

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

func Test_ListUserPermissions(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role to user
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgViewer, orgID, authorization.DomainOrg)
	require.NoError(t, err)

	t.Run("successful list user permissions", func(t *testing.T) {
		req := &pb.ListUserPermissionsRequest{
			UserId:     r.User.String(),
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := ListUserPermissions(ctx, req, authService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Permissions)

		// Should have only read permissions
		hasReadPermission := false
		hasWritePermission := false
		for _, perm := range resp.Permissions {
			if perm.Action == "read" {
				hasReadPermission = true
			}

			if perm.Action == "write" {
				hasWritePermission = true
			}
		}
		assert.True(t, hasReadPermission)
		assert.False(t, hasWritePermission)
	})

	t.Run("invalid request - unspecified domain type", func(t *testing.T) {
		req := &pb.ListUserPermissionsRequest{
			UserId:     r.User.String(),
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
		}

		_, err := ListUserPermissions(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})
}
