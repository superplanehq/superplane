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

func Test_ListAccessibleCanvases(t *testing.T) {
	r := support.Setup(t)
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	canvasID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	// Assign role to user
	err = authService.AssignRole(r.User.String(), authorization.RoleOrgViewer, orgID, authorization.DomainOrg)
	require.NoError(t, err)
	err = authService.AssignRole(r.User.String(), authorization.RoleCanvasViewer, canvasID, authorization.DomainCanvas)
	require.NoError(t, err)

	t.Run("successful list accessible canvases", func(t *testing.T) {
		req := &pb.ListAccessibleCanvasesRequest{
			UserId: r.User.String(),
		}

		resp, err := ListAccessibleCanvases(ctx, req, authService)
		require.NoError(t, err)
		assert.Contains(t, resp.CanvasIds, canvasID)
	})
}
