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

func Test_RemoveInvitation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("removes invitation from organization", func(t *testing.T) {
		// Create an invitation
		invitation := &models.OrganizationInvitation{
			Email:          "test@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		// Remove the invitation
		_, err = RemoveInvitation(ctx, r.AuthService, orgID, invitation.ID.String())
		require.NoError(t, err)

		// Verify the invitation is deleted
		_, err = models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("invitation not found -> error", func(t *testing.T) {
		_, err := RemoveInvitation(ctx, r.AuthService, orgID, uuid.NewString())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "invitation not found", s.Message())
	})
}
