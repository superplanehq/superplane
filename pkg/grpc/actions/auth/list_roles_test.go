package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func Test_ListRoles(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful list roles", func(t *testing.T) {
		req := &pb.ListRolesRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
		}

		resp, err := ListRoles(ctx, req, authService)
		require.NoError(t, err)
		assert.Equal(t, len(resp.Roles), 3) // viewer, admin, owner

		// Should have expected roles
		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.Name
		}
		assert.Contains(t, roleNames, authorization.RoleOrgViewer)
		assert.Contains(t, roleNames, authorization.RoleOrgAdmin)
		assert.Contains(t, roleNames, authorization.RoleOrgOwner)
		assert.Len(t, resp.Roles, 3)
	})

	t.Run("invalid request - unspecified domain type", func(t *testing.T) {
		req := &pb.ListRolesRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
		}

		_, err := ListRoles(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})
}
