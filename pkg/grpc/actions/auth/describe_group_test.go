package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_DescribeGroup(t *testing.T) {
	r := support.Setup(t)
	_ = r // Avoid unused variable warning
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	err = authService.CreateGroup(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description")
	require.NoError(t, err)

	t.Run("successful get organization group", func(t *testing.T) {
		resp, err := DescribeGroup(ctx, models.DomainTypeOrg, orgID, "test-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "test-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
		assert.Equal(t, "org_admin", resp.Group.Spec.Role)
		assert.NotEmpty(t, resp.Group.Spec.DisplayName)
		assert.NotEmpty(t, resp.Group.Spec.Description)
	})

	t.Run("successful get canvas group", func(t *testing.T) {
		canvasID := uuid.New().String()

		// Setup canvas roles and create canvas group with metadata
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas group description")
		require.NoError(t, err)

		resp, err := DescribeGroup(ctx, models.DomainTypeCanvas, canvasID, "canvas-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.Metadata.DomainType)
		assert.Equal(t, canvasID, resp.Group.Metadata.DomainId)
		assert.Equal(t, "canvas_admin", resp.Group.Spec.Role)
		assert.NotEmpty(t, resp.Group.Spec.DisplayName)
		assert.NotEmpty(t, resp.Group.Spec.Description)
	})

	t.Run("group not found", func(t *testing.T) {
		_, err := DescribeGroup(ctx, models.DomainTypeOrg, orgID, "non-existent-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := DescribeGroup(ctx, models.DomainTypeOrg, orgID, "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("different organization - group not found", func(t *testing.T) {
		anotherOrgID := uuid.New().String()
		err := authService.SetupOrganizationRoles(anotherOrgID)
		require.NoError(t, err)

		_, err = DescribeGroup(ctx, models.DomainTypeOrg, anotherOrgID, "test-group", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("get group with viewer role", func(t *testing.T) {
		// Create a group with viewer role
		err = authService.CreateGroup(orgID, models.DomainTypeOrg, "viewer-group", models.RoleOrgViewer, "Viewer Group", "Viewer group description")
		require.NoError(t, err)

		resp, err := DescribeGroup(ctx, models.DomainTypeOrg, orgID, "viewer-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "viewer-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
		assert.Equal(t, "org_viewer", resp.Group.Spec.Role)
	})

	t.Run("get group with owner role", func(t *testing.T) {
		// Create a group with owner role
		err = authService.CreateGroup(orgID, "org", "owner-group", models.RoleOrgOwner, "Owner Group", "Owner group description")
		require.NoError(t, err)

		resp, err := DescribeGroup(ctx, models.DomainTypeOrg, orgID, "owner-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "owner-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
		assert.Equal(t, "org_owner", resp.Group.Spec.Role)
	})
}
