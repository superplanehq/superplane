package actions

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test_AccessibleOrganizations(t *testing.T) {
	r := support.Setup(t)
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Assign roles to user
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgViewer, orgID, authorization.DomainOrg)
	require.NoError(t, err)

	t.Run("successful list accessible organizations", func(t *testing.T) {
		req := &pb.ListAccessibleOrganizationsRequest{
			UserId: r.User.String(),
		}

		resp, err := ListAccessibleOrganizations(ctx, req, authService)
		require.NoError(t, err)
		assert.Contains(t, resp.OrgIds, orgID)
	})
}
