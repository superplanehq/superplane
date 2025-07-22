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

func Test_RemoveUserFromGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "org", "test-group", models.RoleOrgAdmin)
	require.NoError(t, err)

	// Add user to group first
	err = authService.AddUserToGroup(orgID, "org", r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful remove user from group with user ID", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserID:     r.User.String(),
			GroupName:  "test-group",
		}

		err := RemoveUserFromGroup(ctx, req, authService)
		require.NoError(t, err)
	})

	t.Run("successful remove user from group with user email", func(t *testing.T) {
		testEmail := "test-remove-group@example.com"

		// Create user and add to group first
		user := &models.User{
			Name:     testEmail,
			IsActive: false,
		}
		err := user.Create()
		require.NoError(t, err)

		// Create another group to avoid conflicts
		err = authService.CreateGroup(orgID, "org", "test-group-email-remove", models.RoleOrgAdmin)
		require.NoError(t, err)

		err = authService.AddUserToGroup(orgID, "org", user.ID.String(), "test-group-email-remove")
		require.NoError(t, err)

		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserEmail:  testEmail,
			GroupName:  "test-group-email-remove",
		}

		err = RemoveUserFromGroup(ctx, req, authService)
		require.NoError(t, err)
	})

	t.Run("user not found by email", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserEmail:  "nonexistent@example.com",
			GroupName:  "test-group",
		}

		err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserID:     r.User.String(),
			GroupName:  "",
		}

		err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainID:   orgID,
			UserID:     r.User.String(),
			GroupName:  "test-group",
		}

		err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "test-group",
		}

		err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user identifier must be specified")
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserID:     "invalid-uuid",
			GroupName:  "test-group",
		}

		err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("successful canvas group remove user", func(t *testing.T) {
		canvasID := uuid.New().String()

		// Setup canvas roles and create canvas group
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group", models.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.AddUserToGroup(canvasID, "canvas", r.User.String(), "canvas-group")
		require.NoError(t, err)

		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
			UserID:     r.User.String(),
			GroupName:  "canvas-group",
		}

		err = RemoveUserFromGroup(ctx, req, authService)
		require.NoError(t, err)
	})
}
