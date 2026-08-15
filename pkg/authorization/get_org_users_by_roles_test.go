package authorization_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

// GetOrgUsersByRoles resolves every role from one read enforcer. This test
// locks it to the per-role GetOrgUsersForRole method it replaces, so the
// batched path can never silently diverge.
func Test__AuthService_GetOrgUsersByRoles_MatchesGetOrgUsersForRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()
	domainType := models.DomainTypeOrganization

	require.NoError(t, r.AuthService.AssignRole("user-a", models.RoleOrgAdmin, orgID, domainType))
	require.NoError(t, r.AuthService.AssignRole("user-b", models.RoleOrgAdmin, orgID, domainType))
	require.NoError(t, r.AuthService.AssignRole("user-c", models.RoleOrgViewer, orgID, domainType))

	// Includes a role with no members here (owner) to exercise the empty case.
	roleNames := []string{models.RoleOrgAdmin, models.RoleOrgViewer, models.RoleOrgOwner}

	usersByRole, err := r.AuthService.GetOrgUsersByRoles(ctx, orgID, roleNames)
	require.NoError(t, err)

	// Each role's users match the per-role method exactly.
	for _, role := range roleNames {
		want, err := r.AuthService.GetOrgUsersForRole(ctx, role, orgID)
		require.NoError(t, err, "GetOrgUsersForRole(%s)", role)
		assert.ElementsMatch(t, want, usersByRole[role], "users for role %s", role)
	}

	// Concrete expectations so a bug shared by both paths cannot hide. Subset/
	// Contains keep this robust to any roles the org fixture pre-assigns.
	assert.Subset(t, usersByRole[models.RoleOrgAdmin], []string{"user-a", "user-b"})
	assert.Contains(t, usersByRole[models.RoleOrgViewer], "user-c")
}
