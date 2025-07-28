package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
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
	err = authService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", authorization.RoleOrgAdmin)
	require.NoError(t, err)

	t.Run("successful add user to group", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserID:     r.User.String(),
			GroupName:  "test-group",
		}

		err := AddUserToGroup(ctx, req, authService)
		require.NoError(t, err)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			UserID:     r.User.String(),
			GroupName:  "",
		}

		err := AddUserToGroup(ctx, req, authService)
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

		err := AddUserToGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("canvas groups - group does not exist", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
			UserID:     r.User.String(),
			GroupName:  "non-existent-group",
		}

		err = AddUserToGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group non-existent-group does not exist")
	})

	t.Run("successful add user to canvas group", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		// Create a canvas group first
		err = authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-test-group", authorization.RoleCanvasAdmin)
		require.NoError(t, err)

		req := &GroupUserRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
			UserID:     r.User.String(),
			GroupName:  "canvas-test-group",
		}

		err = AddUserToGroup(ctx, req, authService)
		require.NoError(t, err)
	})
}
