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

func Test_DescribeRole(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role description", func(t *testing.T) {
		req := &pb.DescribeRoleRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			Role:       authorization.RoleOrgAdmin,
		}

		resp, err := DescribeRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp.Role)
		assert.NotNil(t, resp.Role.InheritedRole)
		assert.Equal(t, authorization.RoleOrgAdmin, resp.Role.Name)
		assert.Equal(t, authorization.RoleOrgViewer, resp.Role.InheritedRole.Name)
		assert.Len(t, resp.Role.Permissions, 14)
		assert.Len(t, resp.Role.InheritedRole.Permissions, 2)
	})

	t.Run("invalid request - missing domain ID", func(t *testing.T) {
		req := &pb.DescribeRoleRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   "",
			Role:       authorization.RoleOrgAdmin,
		}

		_, err := DescribeRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain ID must be specified")
	})
}
