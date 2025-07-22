package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful role assignment with user ID", func(t *testing.T) {
		req := &pb.AssignRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		resp, err := AssignRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("successful role assignment with user email", func(t *testing.T) {
		testEmail := "test@example.com"
		req := &pb.AssignRoleRequest{
			UserEmail: testEmail,
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		resp, err := AssignRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify user was created
		user, err := models.FindInactiveUserByEmail(testEmail)
		require.NoError(t, err)
		assert.Equal(t, testEmail, user.Name)
		assert.False(t, user.IsActive)
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

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		req := &pb.AssignRoleRequest{
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		_, err := AssignRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user identifier must be specified")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		req := &pb.AssignRoleRequest{
			UserId: "invalid-uuid",
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		_, err := AssignRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
