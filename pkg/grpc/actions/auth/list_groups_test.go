package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func Test_ListGroups(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create some groups first
	err = authService.CreateGroup(orgID, "org", "test-group-1", authorization.RoleOrgAdmin)
	require.NoError(t, err)
	err = authService.CreateGroup(orgID, "org", "test-group-2", authorization.RoleOrgViewer)
	require.NoError(t, err)

	t.Run("successful list groups", func(t *testing.T) {
		req := &GroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
		}

		resp, err := ListGroups(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Groups, 2)

		// Check that groups have the correct structure
		for _, group := range resp.Groups {
			assert.NotEmpty(t, group.Name)
			assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, group.DomainType)
			assert.Equal(t, orgID, group.DomainId)
			// Role is empty for now as noted in TODO
			assert.Equal(t, "", group.Role)
		}

		// Check specific group names
		groupNames := make([]string, len(resp.Groups))
		for i, group := range resp.Groups {
			groupNames[i] = group.Name
		}
		assert.Contains(t, groupNames, "test-group-1")
		assert.Contains(t, groupNames, "test-group-2")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &GroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainID:   orgID,
		}

		_, err := ListGroups(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("successful canvas groups list", func(t *testing.T) {
		canvasID := uuid.New().String()
		
		// Setup canvas roles and create canvas groups
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group-1", authorization.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group-2", authorization.RoleCanvasViewer)
		require.NoError(t, err)
		
		req := &GroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
		}

		resp, err := ListGroups(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Groups, 2)
		
		// Check that groups have the correct structure
		for _, group := range resp.Groups {
			assert.NotEmpty(t, group.Name)
			assert.Equal(t, pb.DomainType_DOMAIN_TYPE_CANVAS, group.DomainType)
			assert.Equal(t, canvasID, group.DomainId)
		}
		
		// Check specific group names
		groupNames := make([]string, len(resp.Groups))
		for i, group := range resp.Groups {
			groupNames[i] = group.Name
		}
		assert.Contains(t, groupNames, "canvas-group-1")
		assert.Contains(t, groupNames, "canvas-group-2")
	})
}