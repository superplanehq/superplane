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

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role assignment", func(t *testing.T) {
		req := &pb.AssignRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       authorization.RoleOrgAdmin,
			},
		}

		resp, err := AssignRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - missing role", func(t *testing.T) {
		req := &pb.AssignRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       "",
			},
		}

		_, err := AssignRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})
}
