package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListRoles(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("successful list roles", func(t *testing.T) {
		resp, err := ListRoles(ctx, models.DomainTypeOrganization, orgID, r.AuthService)
		require.NoError(t, err)
		assert.Equal(t, len(resp.Roles), 3)

		roleNames := make([]string, len(resp.Roles))
		for i, role := range resp.Roles {
			roleNames[i] = role.Metadata.Name
		}
		assert.Contains(t, roleNames, models.RoleOrgOperator)
		assert.Contains(t, roleNames, models.RoleOrgMaintainer)
		assert.Contains(t, roleNames, models.RoleOrgAdmin)
		assert.Len(t, resp.Roles, 3)

		for _, role := range resp.Roles {
			assert.NotEmpty(t, role.Spec.DisplayName, "DisplayName should not be empty for role %s", role.Metadata.Name)
			assert.NotEmpty(t, role.Spec.Description, "Description should not be empty for role %s", role.Metadata.Name)

			switch role.Metadata.Name {
			case models.RoleOrgAdmin:
				assert.Equal(t, "Admin", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Can manage members, billing, and organization settings")
			case models.RoleOrgMaintainer:
				assert.Equal(t, "Maintainer", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Can create and edit automations")
			case models.RoleOrgOperator:
				assert.Equal(t, "Operator", role.Spec.DisplayName)
				assert.Contains(t, role.Spec.Description, "Can create tasks and interact with them")
			}
		}
	})

}
