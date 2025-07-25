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

func Test_ListUserPermissions(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign role to user
	err = authService.AssignRole(r.User.String(), models.RoleOrgViewer, orgID, models.DomainTypeOrg)
	require.NoError(t, err)

	t.Run("successful list user permissions", func(t *testing.T) {
		resp, err := ListUserPermissions(ctx, models.DomainTypeOrg, orgID, r.User.String(), authService)
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
