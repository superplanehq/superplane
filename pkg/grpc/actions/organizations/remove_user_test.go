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

	t.Run("user found with roles -> successfully removes user from organization", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)

		// Remove the user
		_, err := RemoveUser(ctx, r.AuthService, orgID, newUser.ID.String())
		require.NoError(t, err)

		// Verify the user no longer exists
		_, err = models.FindUserByID(orgID, newUser.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// Verify the user no longer has organization roles
		roles, err := r.AuthService.GetUserRolesForOrg(orgID, newUser.ID.String())
		require.NoError(t, err)
		require.Len(t, roles, 0)
	})
}
