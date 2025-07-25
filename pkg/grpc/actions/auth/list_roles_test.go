package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test_ListRoles(t *testing.T) {
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful list roles", func(t *testing.T) {
		resp, err := ListRoles(ctx, models.DomainTypeOrg, orgID, authService)
		require.NoError(t, err)
		assert.Equal(t, len(resp.Roles), 3)

		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.Metadata.Name
		}
		assert.Contains(t, roleNames, models.RoleOrgViewer)
		assert.Contains(t, roleNames, models.RoleOrgAdmin)
		assert.Contains(t, roleNames, models.RoleOrgOwner)
		assert.Len(t, resp.Roles, 3)

		// Test beautiful display names and descriptions for each role
		for _, role := range resp.Roles {
			assert.NotEmpty(t, role.Spec.DisplayName, "DisplayName should not be empty for role %s", role.Metadata.Name)
			assert.NotEmpty(t, role.Spec.Description, "Description should not be empty for role %s", role.Metadata.Name)

			switch role.Metadata.Name {
			case models.RoleOrgOwner:
				assert.Equal(t, "Owner", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Full control over organization settings")
			case models.RoleOrgAdmin:
				assert.Equal(t, "Admin", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Can manage canvases, users, groups, and roles")
			case models.RoleOrgViewer:
				assert.Equal(t, "Viewer", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Read-only access to organization resources")
			}
		}
	})

	t.Run("successful list canvas roles", func(t *testing.T) {
		canvasID := uuid.New().String()
		err := authService.SetupCanvasRoles(canvasID)
		require.NoError(t, err)

		resp, err := ListRoles(ctx, models.DomainTypeCanvas, canvasID, authService)
		require.NoError(t, err)
		assert.Equal(t, len(resp.Roles), 3)

		for _, role := range resp.Roles {
			assert.NotEmpty(t, role.Spec.DisplayName, "DisplayName should not be empty for role %s", role.Metadata.Name)
			assert.NotEmpty(t, role.Spec.Description, "Description should not be empty for role %s", role.Metadata.Name)

			switch role.Metadata.Name {
			case models.RoleCanvasOwner:
				assert.Equal(t, "Owner", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Full control over canvas settings")
			case models.RoleCanvasAdmin:
				assert.Equal(t, "Admin", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Can manage stages, events, connections, and secrets")
			case models.RoleCanvasViewer:
				assert.Equal(t, "Viewer", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Read-only access to canvas resources")
			}
		}
	})

}
