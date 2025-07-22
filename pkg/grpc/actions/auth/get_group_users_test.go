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

func Test_GetGroupUsers(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "org", "test-group", models.RoleOrgAdmin)
	require.NoError(t, err)

	// Add user to group
	err = authService.AddUserToGroup(orgID, "org", r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful get group users", func(t *testing.T) {
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "test-group",
		}

		resp, err := GetGroupUsers(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Equal(t, r.User.String(), resp.Users[0].UserId)
		assert.NotEmpty(t, resp.Users[0].DisplayName)
		assert.NotEmpty(t, resp.Users[0].Email)
		assert.NotEmpty(t, resp.Users[0].RoleAssignments)

		// Check the group object in response
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "test-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
		assert.Equal(t, "org_admin", resp.Group.Role)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "",
		}

		_, err := GetGroupUsers(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainID:   orgID,
			GroupName:  "test-group",
		}

		_, err := GetGroupUsers(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("successful canvas group get users", func(t *testing.T) {
		canvasID := uuid.New().String()

		// Setup canvas roles and create canvas group
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group", models.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.AddUserToGroup(canvasID, "canvas", r.User.String(), "canvas-group")
		require.NoError(t, err)

		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
			GroupName:  "canvas-group",
		}

		resp, err := GetGroupUsers(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Equal(t, r.User.String(), resp.Users[0].UserId)

		// Check the group object in response
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.DomainType)
		assert.Equal(t, canvasID, resp.Group.DomainId)
	})

	t.Run("empty group - no users", func(t *testing.T) {
		// Create another group without users
		err = authService.CreateGroup(orgID, "org", "empty-group", models.RoleOrgViewer)
		require.NoError(t, err)

		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "empty-group",
		}

		resp, err := GetGroupUsers(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Users)

		// Check the group object in response
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "empty-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
	})
}
