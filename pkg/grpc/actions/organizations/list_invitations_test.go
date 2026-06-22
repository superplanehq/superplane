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

	t.Run("no invitations -> returns empty list", func(t *testing.T) {
		response, err := ListInvitations(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Empty(t, response.Invitations)
	})

	t.Run("pending invitations -> returns them", func(t *testing.T) {
		//
		// Create a pending invitation
		//
		invitation := &models.OrganizationInvitation{
			Email:          "invited@example.com",
			OrganizationID: r.Organization.ID,
			InvitedBy:      r.User,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		//
		// List invitations
		//
		response, err := ListInvitations(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.GreaterOrEqual(t, len(response.Invitations), 1)

		//
		// Verify the returned invitation matches what was created
		//
		var found bool
		for _, inv := range response.Invitations {
			if inv.Id == invitation.ID.String() {
				found = true
				assert.Equal(t, r.Organization.ID.String(), inv.OrganizationId)
				assert.Equal(t, "invited@example.com", inv.Email)
				assert.Equal(t, models.InvitationStatePending, inv.State)
				break
			}
		}
		assert.True(t, found, "expected invitation not found")
	})

	t.Run("accepted invitations -> not returned", func(t *testing.T) {
		//
		// Create an accepted invitation
		//
		accepted := &models.OrganizationInvitation{
			Email:          "accepted@example.com",
			OrganizationID: r.Organization.ID,
			InvitedBy:      r.User,
			State:          models.InvitationStateAccepted,
		}
		err := models.SaveInvitation(accepted)
		require.NoError(t, err)

		//
		// List invitations - only pending should appear
		//
		response, err := ListInvitations(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		for _, inv := range response.Invitations {
			assert.Equal(t, models.InvitationStatePending, inv.State,
				"ListInvitations should only return pending invitations")
		}
	})

	t.Run("invitations from different organization -> not visible", func(t *testing.T) {
		//
		// Create a second organization
		//
		org2 := support.CreateOrganization(t, r, r.User)

		//
		// Create a pending invitation in the second organization
		//
		invitation := &models.OrganizationInvitation{
			Email:          "other-org@example.com",
			OrganizationID: org2.ID,
			InvitedBy:      r.User,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		//
		// List invitations for the second organization
		//
		response, err := ListInvitations(ctx, org2.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Invitations, 1)
		assert.Equal(t, "other-org@example.com", response.Invitations[0].Email)

		//
		// Verify the first organization doesn't see the second's invitations
		// (it may have invitations from earlier sub-tests, but not this one)
		//
		response, err = ListInvitations(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		for _, inv := range response.Invitations {
			assert.NotEqual(t, "other-org@example.com", inv.Email,
				"invitations should not leak across organizations")
		}
	})
}
