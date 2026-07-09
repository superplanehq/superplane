package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_DescribeRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("successful role description", func(t *testing.T) {
		resp, err := DescribeRole(ctx, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp.Role)
		assert.NotNil(t, resp.Role.Spec.InheritedRole)
		assert.Equal(t, models.RoleOrgAdmin, resp.Role.Metadata.Name)
		assert.Equal(t, models.RoleOrgViewer, resp.Role.Spec.InheritedRole.Metadata.Name)
		assert.Len(t, resp.Role.Spec.Permissions, 33)
		assert.Len(t, resp.Role.Spec.InheritedRole.Spec.Permissions, 7)
		assert.Equal(t, "Admin", resp.Role.Spec.DisplayName)
		assert.Equal(t, "Viewer", resp.Role.Spec.InheritedRole.Spec.DisplayName)
		assert.Contains(t, resp.Role.Spec.Description, "Can manage canvases, users, groups, and roles")
		assert.Contains(t, resp.Role.Spec.InheritedRole.Spec.Description, "Read-only access to organization resources")
	})
}
