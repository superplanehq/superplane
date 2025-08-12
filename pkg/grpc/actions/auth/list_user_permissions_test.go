package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test_ListUserPermissions(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	// Assign role to user
	require.NoError(t, r.AuthService.AssignRole(r.User.String(), models.RoleOrgViewer, orgID, models.DomainTypeOrganization))

	t.Run("successful list user permissions", func(t *testing.T) {
		resp, err := ListUserPermissions(ctx, models.DomainTypeOrganization, orgID, r.User.String(), r.AuthService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Permissions)

		hasReadPermission := false
		hasWritePermission := false
		for _, perm := range resp.Permissions {
			if perm.Action == "read" {
				hasReadPermission = true
			}

			if perm.Action == "write" {
				hasWritePermission = true
			}
		}
		assert.True(t, hasReadPermission)
		assert.False(t, hasWritePermission)
	})
}
