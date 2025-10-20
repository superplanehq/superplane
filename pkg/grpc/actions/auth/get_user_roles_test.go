package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListUserRoles(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	ctx = authentication.SetOrganizationIdInMetadata(ctx, r.Organization.ID.String())
	orgID := r.Organization.ID.String()

	err := r.AuthService.AssignRole(r.User.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization)
	require.NoError(t, err)

	t.Run("successful get user roles", func(t *testing.T) {
		resp, err := ListUserRoles(ctx, models.DomainTypeOrganization, orgID, r.User.String(), r.AuthService)
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
		_, err := ListUserRoles(ctx, models.DomainTypeOrganization, orgID, "invalid-uuid", r.AuthService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUIDs")
	})
}
