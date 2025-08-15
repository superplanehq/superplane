package organizations

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
	"gorm.io/gorm"
)

func Test_RemoveUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("user not found -> error", func(t *testing.T) {
		_, err := RemoveUser(ctx, r.AuthService, orgID, uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("user found -> removes user from organization", func(t *testing.T) {
		//
		// Add new user to organization, and create new canvases for it.
		//
		newUser := support.CreateUser(t, r, r.Organization.ID)
		canvas1 := support.CreateCanvas(t, r, r.Organization.ID, newUser.ID)
		canvas2 := support.CreateCanvas(t, r, r.Organization.ID, newUser.ID)

		//
		// Remove the user from the organization
		//
		_, err := RemoveUser(ctx, r.AuthService, orgID, newUser.ID.String())
		require.NoError(t, err)

		//
		// Verify the user is soft deleted, and no longer active.
		//
		user, err := models.FindMaybeDeletedUserByID(orgID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, user.DeletedAt)
		_, err = models.FindActiveUserByID(orgID, newUser.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindActiveUserByEmail(orgID, newUser.Email)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)

		//
		// Verify no organization roles exist anymore for that user anymore
		//
		roles, err := r.AuthService.GetUserRolesForOrg(newUser.ID.String(), orgID)
		require.NoError(t, err)
		require.Len(t, roles, 0)

		//
		// Verify that all the canvas access was lost too
		//
		roles, err = r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvas1.ID.String())
		require.NoError(t, err)
		require.Len(t, roles, 0)
		roles, err = r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvas2.ID.String())
		require.NoError(t, err)
		require.Len(t, roles, 0)
	})
}
