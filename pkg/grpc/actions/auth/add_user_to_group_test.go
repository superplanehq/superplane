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

func Test_AddUserToGroup(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "org", "test-group", authorization.RoleOrgAdmin)
	require.NoError(t, err)

	t.Run("successful add user to group", func(t *testing.T) {
		req := &pb.AddUserToGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			UserId:     r.User.String(),
			GroupName:  "test-group",
		}

		resp, err := AddUserToGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.AddUserToGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			UserId:     r.User.String(),
			GroupName:  "",
		}

		_, err := AddUserToGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &pb.AddUserToGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainId:   orgID,
			UserId:     r.User.String(),
			GroupName:  "test-group",
		}

		_, err := AddUserToGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("canvas groups - group does not exist", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		req := &pb.AddUserToGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			UserId:     r.User.String(),
			GroupName:  "non-existent-group",
		}

		_, err = AddUserToGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group non-existent-group does not exist")
	})

	t.Run("successful add user to canvas group", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Create a canvas group first
		err = authService.CreateGroup(canvasID, "canvas", "canvas-test-group", authorization.RoleCanvasAdmin)
		require.NoError(t, err)

		req := &pb.AddUserToGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			UserId:     r.User.String(),
			GroupName:  "canvas-test-group",
		}

		resp, err := AddUserToGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}