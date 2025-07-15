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

func TestGetCanvasUsers(t *testing.T) {
	authService := SetupTestAuthService(t)

	canvasID := uuid.New().String()

	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	err = authService.AssignRole(userID1, "canvas_admin", canvasID, authorization.DomainCanvas)
	require.NoError(t, err)

	err = authService.AssignRole(userID2, "canvas_viewer", canvasID, authorization.DomainCanvas)
	require.NoError(t, err)

	req := &pb.GetCanvasUsersRequest{
		CanvasIdOrName: canvasID,
	}

	resp, err := GetCanvasUsers(context.Background(), req, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Users, 2)

	for _, user := range resp.Users {
		assert.NotEmpty(t, user.UserId)
		assert.NotEmpty(t, user.RoleAssignments)

		for _, roleAssignment := range user.RoleAssignments {
			assert.NotEmpty(t, roleAssignment.RoleName)
			assert.Equal(t, pb.DomainType_DOMAIN_TYPE_CANVAS, roleAssignment.DomainType)
			assert.Equal(t, canvasID, roleAssignment.DomainId)
		}
	}
}

func TestGetCanvasUsersEmptyCanvas(t *testing.T) {
	authService := SetupTestAuthService(t)

	canvasID := uuid.New().String()

	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	req := &pb.GetCanvasUsersRequest{
		CanvasIdOrName: canvasID,
	}

	resp, err := GetCanvasUsers(context.Background(), req, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Users, 0)
}

func TestGetCanvasUsersInvalidCanvasId(t *testing.T) {
	authService := SetupTestAuthService(t)

	req := &pb.GetCanvasUsersRequest{
		CanvasIdOrName: "invalid-uuid",
	}

	resp, err := GetCanvasUsers(context.Background(), req, authService)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "canvas not found")
}
