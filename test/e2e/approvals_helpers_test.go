package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
)

func createOrgApprovalGroup(t *testing.T, orgID uuid.UUID) string {
	t.Helper()

	groupName := fmt.Sprintf("approval_group_%d", time.Now().UnixNano())
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	err = authService.CreateGroup(
		orgID.String(),
		models.DomainTypeOrganization,
		groupName,
		models.RoleOrgAdmin,
		groupName,
		"",
	)
	require.NoError(t, err)

	return groupName
}

func addUserToOrgGroup(t *testing.T, orgID uuid.UUID, email string, groupName string) {
	t.Helper()

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	user, err := models.FindActiveUserByEmail(orgID.String(), email)
	require.NoError(t, err)

	err = authService.AddUserToGroup(orgID.String(), models.DomainTypeOrganization, user.ID.String(), groupName)
	require.NoError(t, err)
}
