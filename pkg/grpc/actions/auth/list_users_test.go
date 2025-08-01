package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func TestGetCanvasUsers(t *testing.T) {
	authService := SetupTestAuthService(t)

	canvasID := uuid.New().String()

	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	err = authService.AssignRole(userID1, "canvas_admin", canvasID, models.DomainTypeCanvas)
	require.NoError(t, err)

	err = authService.AssignRole(userID2, "canvas_viewer", canvasID, models.DomainTypeCanvas)
	require.NoError(t, err)

	resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, canvasID, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Users, 2)

	for _, user := range resp.Users {
		assert.NotEmpty(t, user.Metadata.Id)
		assert.NotEmpty(t, user.Status.RoleAssignments)

		assert.False(t, user.Status.IsActive)

		for _, roleAssignment := range user.Status.RoleAssignments {
			assert.NotEmpty(t, roleAssignment.RoleName)
			assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, roleAssignment.DomainType)
			assert.Equal(t, canvasID, roleAssignment.DomainId)
		}
	}
}

func TestGetCanvasUsersEmptyCanvas(t *testing.T) {
	authService := SetupTestAuthService(t)

	canvasID := uuid.New().String()

	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, canvasID, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Users, 0)
}

func TestGetCanvasUsersWithActiveUser(t *testing.T) {
	authService := SetupTestAuthService(t)

	canvasID := uuid.New().String()

	err := authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	// Create an active user
	user := &models.User{
		Name:     "Active Canvas User",
		IsActive: true,
	}
	err = user.Create()
	require.NoError(t, err)

	// Assign role to the active user
	err = authService.AssignRole(user.ID.String(), "canvas_admin", canvasID, models.DomainTypeCanvas)
	require.NoError(t, err)

	resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, canvasID, authService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have 1 user
	assert.Len(t, resp.Users, 1)

	// Check that the active user is properly returned
	activeUser := resp.Users[0]
	assert.Equal(t, user.ID.String(), activeUser.Metadata.Id)
	assert.True(t, activeUser.Status.IsActive)
	assert.Equal(t, "Active Canvas User", activeUser.Spec.DisplayName)
	assert.NotEmpty(t, activeUser.Status.RoleAssignments)

	// Check role assignment details
	assert.Equal(t, "canvas_admin", activeUser.Status.RoleAssignments[0].RoleName)
	assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, activeUser.Status.RoleAssignments[0].DomainType)
	assert.Equal(t, canvasID, activeUser.Status.RoleAssignments[0].DomainId)
}
