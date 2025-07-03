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

func Test_RemoveUserFromGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "org", "test-group", authorization.RoleOrgAdmin)
	require.NoError(t, err)

	// Add user to group first
	err = authService.AddUserToGroup(orgID, "org", r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful remove user from group", func(t *testing.T) {
		req := &pb.RemoveUserFromGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			UserId:     r.User.String(),
			GroupName:  "test-group",
		}

		resp, err := RemoveUserFromGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.RemoveUserFromGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			UserId:     r.User.String(),
			GroupName:  "",
		}

		_, err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &pb.RemoveUserFromGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
			UserId:     r.User.String(),
			GroupName:  "test-group",
		}

		_, err := RemoveUserFromGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("successful canvas group remove user", func(t *testing.T) {
		canvasID := uuid.New().String()
		
		// Setup canvas roles and create canvas group
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group", authorization.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.AddUserToGroup(canvasID, "canvas", r.User.String(), "canvas-group")
		require.NoError(t, err)
		
		req := &pb.RemoveUserFromGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			UserId:     r.User.String(),
			GroupName:  "canvas-group",
		}

		resp, err := RemoveUserFromGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}