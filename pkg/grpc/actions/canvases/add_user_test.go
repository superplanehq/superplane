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

func Test_AddUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("user not found -> error", func(t *testing.T) {
		_, err := AddUser(ctx, r.AuthService, orgID, canvasID, uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("user found -> successfully adds user to canvas", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		response, err := AddUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify the user was assigned the canvas viewer role
		roles, err := r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvasID)
		require.NoError(t, err)
		require.Len(t, roles, 1)
		assert.Equal(t, models.RoleCanvasViewer, roles[0].Name)
	})
}
