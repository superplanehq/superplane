package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_RemoveUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("user not found -> error", func(t *testing.T) {
		_, err := RemoveUser(ctx, r.AuthService, orgID, canvasID, uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("removes user with viewer role from canvas", func(t *testing.T) {
		// Create and new user to canvas
		newUser := support.CreateUser(t, r, r.Organization.ID)
		_, err := AddUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)

		// Remove the user
		response, err := RemoveUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify the user no longer has canvas roles
		roles, err := r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvasID)
		require.NoError(t, err)
		require.Empty(t, roles)
	})

	t.Run("removes user with admin role from canvas", func(t *testing.T) {
		// Create and new user to canvas
		newUser := support.CreateUser(t, r, r.Organization.ID)
		err := r.AuthService.AssignRole(newUser.ID.String(), models.RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		_, err = AddUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)

		// Remove the user
		response, err := RemoveUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify the user no longer has canvas roles
		roles, err := r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvasID)
		require.NoError(t, err)
		require.Empty(t, roles)
	})
}
