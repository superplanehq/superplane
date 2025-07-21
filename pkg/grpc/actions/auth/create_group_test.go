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
)

func Test_CreateGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful group creation", func(t *testing.T) {
		req := &CreateGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "test-group",
			Role:       authorization.RoleOrgAdmin,
		}

		resp, err := CreateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if group was created
		groups, err := authService.GetGroups(orgID, models.DomainTypeOrganization)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")
		assert.Len(t, groups, 1)
	})

	t.Run("successful canvas group creation", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		req := &CreateGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
			GroupName:  "canvas-group",
			Role:       authorization.RoleCanvasAdmin,
		}

		resp, err := CreateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "canvas-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.DomainType)
		assert.Equal(t, canvasID, resp.Group.DomainId)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &CreateGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "",
			Role:       authorization.RoleOrgAdmin,
		}

		_, err := CreateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})
}
