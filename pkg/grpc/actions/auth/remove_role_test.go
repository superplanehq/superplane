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

func Test_RemoveRole(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role first
	err = authService.AssignRole(r.User.String(), models.RoleOrgAdmin, orgID, models.DomainOrg)
	require.NoError(t, err)

	t.Run("successful role removal with user ID", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		resp, err := RemoveRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("successful role removal with user email", func(t *testing.T) {
		testEmail := "test-remove@example.com"

		// Create user and assign role first
		user := &models.User{
			Name:     testEmail,
			IsActive: false,
		}
		err := user.Create()
		require.NoError(t, err)

		accountProvider := &models.AccountProvider{
			Provider: "github",
			UserID:   user.ID,
			Email:    testEmail,
		}
		err = accountProvider.Create()
		require.NoError(t, err)

		err = authService.AssignRole(user.ID.String(), models.RoleOrgAdmin, orgID, models.DomainOrg)
		require.NoError(t, err)

		req := &pb.RemoveRoleRequest{
			UserEmail: testEmail,
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		resp, err := RemoveRole(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("user not found by email", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			UserEmail: "nonexistent@example.com",
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		_, err := RemoveRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - unspecified domain type", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			UserId: r.User.String(),
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		_, err := RemoveRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		_, err := RemoveRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID or Email")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		req := &pb.RemoveRoleRequest{
			UserId: "invalid-uuid",
			RoleAssignment: &pb.RoleAssignment{
				DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
				DomainId:   orgID,
				Role:       models.RoleOrgAdmin,
			},
		}

		_, err := RemoveRole(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}
