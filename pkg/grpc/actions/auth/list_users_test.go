package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func TestGetCanvasUsers(t *testing.T) {
	r := support.Setup(t)
	canvasID := r.Canvas.ID.String()

	userID1 := uuid.New().String()
	userID2 := uuid.New().String()

	require.NoError(t, r.AuthService.AssignRole(userID1, "canvas_admin", canvasID, models.DomainTypeCanvas))
	require.NoError(t, r.AuthService.AssignRole(userID2, "canvas_viewer", canvasID, models.DomainTypeCanvas))

	resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, canvasID, r.AuthService)
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
	r := support.Setup(t)
	canvasID := r.Canvas.ID.String()

	resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, canvasID, r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Len(t, resp.Users, 0)
}

func TestGetCanvasUsersWithActiveUser(t *testing.T) {
	r := support.Setup(t)
	canvasID := r.Canvas.ID.String()

	// Assign role to the active user
	require.NoError(t, r.AuthService.AssignRole(r.User.String(), "canvas_admin", canvasID, models.DomainTypeCanvas))

	resp, err := ListUsers(context.Background(), models.DomainTypeCanvas, canvasID, r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 1)

	activeUser := resp.Users[0]
	assert.Equal(t, r.User.String(), activeUser.Metadata.Id)
	assert.True(t, activeUser.Status.IsActive)
	assert.Equal(t, "Active Canvas User", activeUser.Spec.DisplayName)
	assert.NotEmpty(t, activeUser.Status.RoleAssignments)
	assert.Equal(t, "canvas_admin", activeUser.Status.RoleAssignments[0].RoleName)
	assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, activeUser.Status.RoleAssignments[0].DomainType)
	assert.Equal(t, canvasID, activeUser.Status.RoleAssignments[0].DomainId)
}
