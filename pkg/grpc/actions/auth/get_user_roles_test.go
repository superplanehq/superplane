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

func Test_GetUserRoles(t *testing.T) {
	r := support.Setup(t)
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role to user
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgAdmin, orgID, authorization.DomainOrg)
	require.NoError(t, err)

	t.Run("successful get user roles", func(t *testing.T) {
		req := &pb.GetUserRolesRequest{
			UserId:     r.User.String(),
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := GetUserRoles(ctx, req, authService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Roles)

		// Should have at least the assigned role
		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.Name
		}
		assert.Contains(t, roleNames, authorization.RoleOrgAdmin)
		assert.Contains(t, roleNames, authorization.RoleOrgViewer)
		assert.NotContains(t, roleNames, authorization.RoleOrgOwner)
	})

	t.Run("invalid request - invalid UUID", func(t *testing.T) {
		req := &pb.GetUserRolesRequest{
			UserId:     "invalid-uuid",
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		_, err := GetUserRoles(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUIDs")
	})
}
