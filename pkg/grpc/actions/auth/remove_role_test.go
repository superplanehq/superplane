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

func Test_RemoveRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role first
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgAdmin, orgID, authorization.DomainOrg)
	require.NoError(t, err)

	t.Run("successful role removal", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       authorization.RoleOrgAdmin,
			},
		}

		resp, err := RemoveRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - unspecified domain type", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
				DomainId:   orgID,
				Role:       authorization.RoleOrgAdmin,
			},
		}

		_, err := RemoveRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})
}
