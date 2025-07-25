package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListUserRoles(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	err = authService.AssignRole(r.User.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
	require.NoError(t, err)

	t.Run("successful get user roles", func(t *testing.T) {
		resp, err := ListUserRoles(ctx, models.DomainTypeOrganization, orgID, r.User.String(), authService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Roles)

		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.Metadata.Name
		}
		assert.Contains(t, roleNames, models.RoleOrgAdmin)
		assert.Contains(t, roleNames, models.RoleOrgViewer)
		assert.NotContains(t, roleNames, models.RoleOrgOwner)
	})

	t.Run("invalid request - invalid UUID", func(t *testing.T) {
		_, err := ListUserRoles(ctx, models.DomainTypeOrganization, orgID, "invalid-uuid", authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUIDs")
	})
}
