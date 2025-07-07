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

func Test_GetGroupUsers(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "org", "test-group", authorization.RoleOrgAdmin)
	require.NoError(t, err)

	// Add user to group
	err = authService.AddUserToGroup(orgID, "org", r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful get group users", func(t *testing.T) {
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "test-group",
		}

		resp, err := GetGroupUsers(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.UserIds, 1)
		assert.Contains(t, resp.UserIds, r.User.String())

		// Check the group object in response
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "test-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
		// Role is empty for now as noted in TODO
		assert.Equal(t, "", resp.Group.Role)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "",
		}

		_, err := GetGroupUsers(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
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
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group", authorization.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.AddUserToGroup(canvasID, "canvas", r.User.String(), "canvas-group")
		require.NoError(t, err)
		
		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			GroupName:  "canvas-group",
		}

		resp, err := GetGroupUsers(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.UserIds, 1)
		assert.Contains(t, resp.UserIds, r.User.String())
		
		// Check the group object in response
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.DomainType)
		assert.Equal(t, canvasID, resp.Group.DomainId)
	})

	t.Run("empty group - no users", func(t *testing.T) {
		// Create another group without users
		err = authService.CreateGroup(orgID, "org", "empty-group", authorization.RoleOrgViewer)
		require.NoError(t, err)

		req := &GetGroupUsersRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "empty-group",
		}

		resp, err := GetGroupUsers(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.UserIds)

		// Check the group object in response
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "empty-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
	})
}