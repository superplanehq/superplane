package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_DescribeGroup(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	err := r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description")
	require.NoError(t, err)

	t.Run("successful get organization group", func(t *testing.T) {
		resp, err := DescribeGroup(ctx, models.DomainTypeOrganization, orgID, "test-group", r.AuthService)
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
		err = r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas group description")
		require.NoError(t, err)

		resp, err := DescribeGroup(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), "canvas-group", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.Metadata.DomainType)
		assert.Equal(t, r.Canvas.ID.String(), resp.Group.Metadata.DomainId)
		assert.Equal(t, "canvas_admin", resp.Group.Spec.Role)
		assert.NotEmpty(t, resp.Group.Spec.DisplayName)
		assert.NotEmpty(t, resp.Group.Spec.Description)
	})

	t.Run("group not found", func(t *testing.T) {
		_, err := DescribeGroup(ctx, models.DomainTypeOrganization, orgID, "non-existent-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := DescribeGroup(ctx, models.DomainTypeOrganization, orgID, "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("different organization - group not found", func(t *testing.T) {
		anotherOrg, err := models.CreateOrganization("test-org", "Test Organization")
		require.NoError(t, err)
		tx := database.Conn().Begin()
		err = r.AuthService.SetupOrganization(tx, orgID, r.User.String())
		if !assert.NoError(t, err) {
			tx.Rollback()
			t.FailNow()
		}

		err = tx.Commit().Error
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		_, err = DescribeGroup(ctx, models.DomainTypeOrganization, anotherOrg.ID.String(), "test-group", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group not found")
	})

	t.Run("get group with viewer role", func(t *testing.T) {
		// Create a group with viewer role
		err = r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "viewer-group", models.RoleOrgViewer, "Viewer Group", "Viewer group description")
		require.NoError(t, err)

		resp, err := DescribeGroup(ctx, models.DomainTypeOrganization, orgID, "viewer-group", r.AuthService)
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
		err = r.AuthService.CreateGroup(orgID, "org", "owner-group", models.RoleOrgOwner, "Owner Group", "Owner group description")
		require.NoError(t, err)

		resp, err := DescribeGroup(ctx, models.DomainTypeOrganization, orgID, "owner-group", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "owner-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
		assert.Equal(t, "org_owner", resp.Group.Spec.Role)
	})
}
