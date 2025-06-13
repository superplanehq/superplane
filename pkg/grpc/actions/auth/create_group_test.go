package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func Test_CreateGroup(t *testing.T) {
	authService := setupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	t.Run("successful group creation", func(t *testing.T) {
		req := &pb.CreateGroupRequest{
			OrgId:     orgID,
			GroupName: "test-group",
			Role:      authorization.RoleOrgAdmin,
		}

		resp, err := CreateGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Check if group was created
		groups, err := authService.GetGroups(orgID)
		require.NoError(t, err)
		assert.Contains(t, groups, "test-group")
		assert.Len(t, groups, 1)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.CreateGroupRequest{
			OrgId:     orgID,
			GroupName: "",
			Role:      authorization.RoleOrgAdmin,
		}

		_, err := CreateGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})
}
