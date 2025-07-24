package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
)

func Test_CreateGroup(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful group creation", func(t *testing.T) {
		req := &pb.CreateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "test-group",
			Role:       models.RoleOrgAdmin,
		}

		resp, err := CreateGroup(ctx, "org", orgID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if group was created
		groups, err := authService.GetGroups(orgID, models.DomainTypeOrg)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")
		assert.Len(t, groups, 1)
	})

	t.Run("successful canvas group creation", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		req := &pb.CreateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_CANVAS,
			DomainId:   canvasID,
			GroupName:  "canvas-group",
			Role:       models.RoleCanvasAdmin,
		}

		resp, err := CreateGroup(ctx, "canvas", canvasID, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "canvas-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.Metadata.DomainType)
		assert.Equal(t, canvasID, resp.Group.Metadata.DomainId)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.CreateGroupRequest{
			DomainType: pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainId:   orgID,
			GroupName:  "",
			Role:       models.RoleOrgAdmin,
		}

		_, err := CreateGroup(ctx, "org", orgID, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})
}
