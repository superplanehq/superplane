package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListInvitations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("returns empty list when no pending invitations", func(t *testing.T) {
		response, err := ListInvitations(ctx, orgID)
		require.NoError(t, err)
		assert.Empty(t, response.Invitations)
	})

	t.Run("returns only pending invitations", func(t *testing.T) {
		// Create a pending invitation
		_, err := CreateInvitation(ctx, r.AuthService, orgID, "pending@example.com")
		require.NoError(t, err)

		// Create an account with an accepted invitation
		account, err := models.CreateAccount(support.RandomName("account")+"@example.com", support.RandomName("user"))
		require.NoError(t, err)
		_, err = CreateInvitation(ctx, r.AuthService, orgID, account.Email)
		require.NoError(t, err)

		response, err := ListInvitations(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, response.Invitations, 1)
		assert.Equal(t, "pending@example.com", response.Invitations[0].Email)
		assert.Equal(t, models.InvitationStatePending, response.Invitations[0].State)
		assert.Equal(t, orgID, response.Invitations[0].OrganizationId)
	})

	t.Run("removed invitation no longer appears in list", func(t *testing.T) {
		pending, err := models.ListInvitationsInState(orgID, models.InvitationStatePending)
		require.NoError(t, err)
		require.NotEmpty(t, pending)
		invitation := pending[0]

		_, err = RemoveInvitation(ctx, r.AuthService, orgID, invitation.ID.String())
		require.NoError(t, err)

		response, err := ListInvitations(ctx, orgID)
		require.NoError(t, err)
		for _, inv := range response.Invitations {
			assert.NotEqual(t, invitation.ID.String(), inv.Id)
		}
	})
}
