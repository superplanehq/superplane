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

func Test_GetGroup(t *testing.T) {
	r := support.Setup(t)
	_ = r // Avoid unused variable warning
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "org", "test-group", authorization.RoleOrgAdmin)
	require.NoError(t, err)

	t.Run("successful get organization group", func(t *testing.T) {
		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "test-group",
		}

		resp, err := GetGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "test-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
		assert.Equal(t, "org_admin", resp.Group.Role)
		assert.NotEmpty(t, resp.Group.DisplayName)
		assert.NotEmpty(t, resp.Group.Description)
	})

	t.Run("successful get canvas group", func(t *testing.T) {
		canvasID := uuid.New().String()

		// Setup canvas roles and create canvas group
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, "canvas", "canvas-group", authorization.RoleCanvasAdmin)
		require.NoError(t, err)

		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
			DomainID:   canvasID,
			GroupName:  "canvas-group",
		}

		resp, err := GetGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.DomainType)
		assert.Equal(t, canvasID, resp.Group.DomainId)
		assert.Equal(t, "canvas_admin", resp.Group.Role)
		assert.NotEmpty(t, resp.Group.DisplayName)
		assert.NotEmpty(t, resp.Group.Description)
	})

	t.Run("group not found", func(t *testing.T) {
		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "non-existent-group",
		}

		_, err := GetGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "",
		}

		_, err := GetGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("invalid request - missing domain type", func(t *testing.T) {
		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_UNSPECIFIED,
			DomainID:   orgID,
			GroupName:  "test-group",
		}

		_, err := GetGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain type must be specified")
	})

	t.Run("invalid request - invalid domain ID", func(t *testing.T) {
		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   "invalid-uuid",
			GroupName:  "test-group",
		}

		_, err := GetGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid domain ID")
	})

	t.Run("different organization - group not found", func(t *testing.T) {
		anotherOrgID := uuid.New().String()
		err := authService.SetupOrganizationRoles(anotherOrgID)
		require.NoError(t, err)

		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   anotherOrgID,
			GroupName:  "test-group", // This group exists in the first org, not this one
		}

		_, err = GetGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("get group with viewer role", func(t *testing.T) {
		// Create a group with viewer role
		err = authService.CreateGroup(orgID, "org", "viewer-group", authorization.RoleOrgViewer)
		require.NoError(t, err)

		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "viewer-group",
		}

		resp, err := GetGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "viewer-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
		assert.Equal(t, "org_viewer", resp.Group.Role)
	})

	t.Run("get group with owner role", func(t *testing.T) {
		// Create a group with owner role
		err = authService.CreateGroup(orgID, "org", "owner-group", authorization.RoleOrgOwner)
		require.NoError(t, err)

		req := &GetGroupRequest{
			DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
			DomainID:   orgID,
			GroupName:  "owner-group",
		}

		resp, err := GetGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "owner-group", resp.Group.Name)
		assert.Equal(t, pb.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.DomainType)
		assert.Equal(t, orgID, resp.Group.DomainId)
		assert.Equal(t, "org_owner", resp.Group.Role)
	})
}