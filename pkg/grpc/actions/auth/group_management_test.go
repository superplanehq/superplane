package auth

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

func Test_GroupManagement(t *testing.T) {
	r := support.Setup(t)
	authService := SetupTestAuthService(t)
	ctx := context.Background()

	orgID := uuid.New().String()
	err := authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	// Create a group first
	err = authService.CreateGroup(orgID, "test-group", authorization.RoleOrgAdmin)
	require.NoError(t, err)

	t.Run("successful add user to group", func(t *testing.T) {
		req := &pb.AddUserToGroupRequest{
			OrgId:     orgID,
			UserId:    r.User.String(),
			GroupName: "test-group",
		}

		resp, err := AddUserToGroup(ctx, req, authService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - missing group name", func(t *testing.T) {
		req := &pb.AddUserToGroupRequest{
			OrgId:     orgID,
			UserId:    r.User.String(),
			GroupName: "",
		}

		_, err := AddUserToGroup(ctx, req, authService)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "group name must be specified")
	})
}
