package authorization_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

// Locks GetGroupsWithDetails to GetGroupRole and GetGroupUsers.
func Test__AuthService_GetGroupsWithDetails_MatchesPerGroupMethods(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()
	domainType := models.DomainTypeOrganization

	require.NoError(t, r.AuthService.CreateGroup(orgID, domainType, "admins", models.RoleOrgAdmin, "Admins", "Admin group"))
	require.NoError(t, r.AuthService.CreateGroup(orgID, domainType, "viewers", models.RoleOrgViewer, "Viewers", "Viewer group"))
	// A group with no members exercises the empty-members path.
	require.NoError(t, r.AuthService.CreateGroup(orgID, domainType, "empty", models.RoleOrgViewer, "Empty", "No members"))

	require.NoError(t, r.AuthService.AddUserToGroup(orgID, domainType, "user-a", "admins"))
	require.NoError(t, r.AuthService.AddUserToGroup(orgID, domainType, "user-b", "admins"))
	require.NoError(t, r.AuthService.AddUserToGroup(orgID, domainType, "user-c", "viewers"))

	details, err := r.AuthService.GetGroupsWithDetails(ctx, orgID, domainType)
	require.NoError(t, err)

	// Same set of groups as GetGroups.
	expectedNames, err := r.AuthService.GetGroups(ctx, orgID, domainType)
	require.NoError(t, err)
	actualNames := make([]string, len(details))
	for i, d := range details {
		actualNames[i] = d.Name
	}
	assert.ElementsMatch(t, expectedNames, actualNames)

	// Each group's role and members match the per-group methods exactly.
	for _, detail := range details {
		wantRole, err := r.AuthService.GetGroupRole(ctx, orgID, domainType, detail.Name)
		require.NoError(t, err, "GetGroupRole(%s)", detail.Name)
		assert.Equal(t, wantRole, detail.Role, "role for group %s", detail.Name)

		wantMembers, err := r.AuthService.GetGroupUsers(ctx, orgID, domainType, detail.Name)
		require.NoError(t, err, "GetGroupUsers(%s)", detail.Name)
		assert.ElementsMatch(t, wantMembers, detail.Members, "members for group %s", detail.Name)
	}

	// Spot-check the concrete expectations so a bug in both paths can't hide.
	byName := make(map[string]int)
	for _, detail := range details {
		byName[detail.Name] = len(detail.Members)
	}
	assert.Equal(t, 2, byName["admins"])
	assert.Equal(t, 1, byName["viewers"])
	assert.Equal(t, 0, byName["empty"])
}
