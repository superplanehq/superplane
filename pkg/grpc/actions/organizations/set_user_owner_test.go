package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test_SetUserOwner(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("sets owner flag on another member", func(t *testing.T) {
		member := support.CreateUser(t, r, r.Organization.ID)
		require.False(t, member.IsOwner)

		_, err := SetUserOwner(ctx, orgID, member.ID.String(), true)
		require.NoError(t, err)

		updated, err := models.FindActiveUserByID(orgID, member.ID.String())
		require.NoError(t, err)
		assert.True(t, updated.IsOwner)
	})

	t.Run("refuses to clear the last owner", func(t *testing.T) {
		owner, err := models.FindActiveUserByID(orgID, r.User.String())
		require.NoError(t, err)
		require.True(t, owner.IsOwner)

		_, err = SetUserOwner(ctx, orgID, r.User.String(), false)
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "cannot remove the last organization owner", msg)
	})

	t.Run("clears owner flag when another owner remains", func(t *testing.T) {
		member := support.CreateUser(t, r, r.Organization.ID)
		_, err := SetUserOwner(ctx, orgID, member.ID.String(), true)
		require.NoError(t, err)

		_, err = SetUserOwner(ctx, orgID, r.User.String(), false)
		require.NoError(t, err)

		updated, err := models.FindActiveUserByID(orgID, r.User.String())
		require.NoError(t, err)
		assert.False(t, updated.IsOwner)

		require.NoError(t, models.SetUserIsOwner(database.DB(ctx), r.User, true))
	})

	t.Run("refuses API key owners", func(t *testing.T) {
		apiKey, err := models.CreateAPIKey(database.Conn(), r.Organization.ID, "bot", nil, r.User, nil, nil)
		require.NoError(t, err)

		_, err = SetUserOwner(ctx, orgID, apiKey.ID.String(), true)
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "API keys cannot be organization owners", msg)
	})
}
