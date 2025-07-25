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

func Test_ListGroupUsers(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, models.DomainTypeOrg, "test-group", models.RoleOrgAdmin)
	require.NoError(t, err)

	// Add user to group
	err = authService.AddUserToGroup(orgID, models.DomainTypeOrg, r.User.String(), "test-group")
	require.NoError(t, err)

	t.Run("successful get group users", func(t *testing.T) {

		resp, err := ListGroupUsers(ctx, models.DomainTypeOrg, orgID, "test-group", authService)
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
		_, err := ListGroupUsers(ctx, models.DomainTypeOrg, orgID, "", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})

	t.Run("successful canvas group get users", func(t *testing.T) {
		canvasID := uuid.New().String()

		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)
		err = authService.CreateGroup(canvasID, models.DomainTypeCanvas, "canvas-group", models.RoleCanvasAdmin)
		require.NoError(t, err)
		err = authService.AddUserToGroup(canvasID, models.DomainTypeCanvas, r.User.String(), "canvas-group")
		require.NoError(t, err)

		resp, err := ListGroupUsers(ctx, models.DomainTypeCanvas, canvasID, "canvas-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Equal(t, r.User.String(), resp.Users[0].Metadata.Id)

		assert.NotNil(t, resp.Group)
		assert.Equal(t, "canvas-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, resp.Group.Metadata.DomainType)
		assert.Equal(t, canvasID, resp.Group.Metadata.DomainId)
	})

	t.Run("empty group - no users", func(t *testing.T) {
		err = authService.CreateGroup(orgID, models.DomainTypeOrg, "empty-group", models.RoleOrgViewer)
		require.NoError(t, err)

		resp, err := ListGroupUsers(ctx, models.DomainTypeOrg, orgID, "empty-group", authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.Users)

		assert.NotNil(t, resp.Group)
		assert.Equal(t, "empty-group", resp.Group.Metadata.Name)
		assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, resp.Group.Metadata.DomainType)
		assert.Equal(t, orgID, resp.Group.Metadata.DomainId)
	})
}
