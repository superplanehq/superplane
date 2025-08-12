package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListGroupUsers(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	// Create a group first
	require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "test-group", models.RoleOrgAdmin, "Test Group", "Test group description"))
	require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, r.User.String(), "test-group"))

	t.Run("successful get group users", func(t *testing.T) {
		resp, err := ListGroupUsers(ctx, models.DomainTypeOrganization, orgID, "test-group", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Equal(t, r.User.String(), resp.Users[0].Metadata.Id)
		assert.NotEmpty(t, resp.Users[0].Spec.DisplayName)
		assert.NotEmpty(t, resp.Users[0].Metadata.Email)
		assert.NotEmpty(t, resp.Users[0].Status.RoleAssignments)

		assert.NotNil(t, resp.Group)
		assert.Equal(t, "test-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
		assert.Equal(t, "org_admin", resp.Group.Spec.Role)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		_, err := ListGroupUsers(ctx, models.DomainTypeOrganization, orgID, "", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("successful canvas group get users", func(t *testing.T) {
		require.NoError(t, r.AuthService.CreateGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin, "Canvas Group", "Canvas group description"))
		require.NoError(t, r.AuthService.AddUserToGroup(r.Canvas.ID.String(), models.DomainTypeCanvas, r.User.String(), "canvas-group"))

		resp, err := ListGroupUsers(ctx, models.DomainTypeCanvas, r.Canvas.ID.String(), "canvas-group", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Equal(t, r.User.String(), resp.Users[0].Metadata.Id)
		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.Metadata.DomainType)
		assert.Equal(t, r.Canvas.ID.String(), resp.Group.Metadata.DomainId)
	})

	t.Run("empty group - no users", func(t *testing.T) {
		require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "empty-group", models.RoleOrgViewer, "Empty Group", "Empty group description"))

		resp, err := ListGroupUsers(ctx, models.DomainTypeOrganization, orgID, "empty-group", r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Users)

		assert.NotNil(t, resp.Group)
		assert.Equal(t, "empty-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
	})
}
