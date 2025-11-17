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
	"gorm.io/datatypes"
)

func Test_UpdateInvitation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("updates invitation with valid canvas IDs", func(t *testing.T) {
		// Create a canvas
		canvas, err := models.CreateCanvas(r.User, r.Organization.ID, "Test Canvas", "")
		require.NoError(t, err)

		// Create an invitation
		invitation := &models.OrganizationInvitation{
			Email:          "test@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err = models.SaveInvitation(invitation)
		require.NoError(t, err)

		canvasIDs := []string{canvas.ID.String()}

		// Update the invitation
		response, err := UpdateInvitation(ctx, r.AuthService, orgID, invitation.ID.String(), canvasIDs)
		require.NoError(t, err)
		assert.Equal(t, invitation.ID.String(), response.Invitation.Id)

		// Verify the invitation was updated
		updatedInvitation, err := models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)

		savedCanvasIDs := updatedInvitation.CanvasIDs.Data()
		assert.Equal(t, canvasIDs, savedCanvasIDs)
	})

	t.Run("updates invitation with empty canvas IDs", func(t *testing.T) {
		// Create an invitation
		invitation := &models.OrganizationInvitation{
			Email:          "test2@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		canvasIDs := []string{}

		// Update the invitation
		response, err := UpdateInvitation(ctx, r.AuthService, orgID, invitation.ID.String(), canvasIDs)
		require.NoError(t, err)
		assert.Equal(t, invitation.ID.String(), response.Invitation.Id)

		// Verify the invitation was updated
		updatedInvitation, err := models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)

		savedCanvasIDs := updatedInvitation.CanvasIDs.Data()
		assert.Equal(t, canvasIDs, savedCanvasIDs)
	})

	t.Run("updates invitation with multiple canvas IDs", func(t *testing.T) {
		// Create multiple canvases
		canvas1, err := models.CreateCanvas(r.User, r.Organization.ID, "Test Canvas 1", "")
		require.NoError(t, err)

		canvas2, err := models.CreateCanvas(r.User, r.Organization.ID, "Test Canvas 2", "")
		require.NoError(t, err)

		// Create an invitation
		invitation := &models.OrganizationInvitation{
			Email:          "test3@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
			CanvasIDs:      datatypes.NewJSONType([]string{}),
		}
		err = models.SaveInvitation(invitation)
		require.NoError(t, err)

		canvasIDs := []string{canvas1.ID.String(), canvas2.ID.String()}

		// Update the invitation
		response, err := UpdateInvitation(ctx, r.AuthService, orgID, invitation.ID.String(), canvasIDs)
		require.NoError(t, err)
		assert.Equal(t, invitation.ID.String(), response.Invitation.Id)

		// Verify the invitation was updated
		updatedInvitation, err := models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)

		savedCanvasIDs := updatedInvitation.CanvasIDs.Data()
		assert.ElementsMatch(t, canvasIDs, savedCanvasIDs)
	})

	t.Run("invitation not found -> error", func(t *testing.T) {
		_, err := UpdateInvitation(ctx, r.AuthService, orgID, uuid.NewString(), []string{})
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "invitation not found", s.Message())
	})

	t.Run("invalid canvas ID format -> error", func(t *testing.T) {
		// Create an invitation
		invitation := &models.OrganizationInvitation{
			Email:          "test4@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		canvasIDs := []string{"invalid-uuid"}

		_, err = UpdateInvitation(ctx, r.AuthService, orgID, invitation.ID.String(), canvasIDs)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		// Create an invitation
		invitation := &models.OrganizationInvitation{
			Email:          "test5@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		canvasIDs := []string{uuid.NewString()}

		_, err = UpdateInvitation(ctx, r.AuthService, orgID, invitation.ID.String(), canvasIDs)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("canvas belongs to different organization -> error", func(t *testing.T) {
		// Create another organization
		otherOrg, err := models.CreateOrganization("Other Org", "")
		require.NoError(t, err)

		// Create a canvas in the other organization
		canvas, err := models.CreateCanvas(r.User, otherOrg.ID, "Canvas in Other Org", "")
		require.NoError(t, err)

		// Create an invitation in the original organization
		invitation := &models.OrganizationInvitation{
			Email:          "test6@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err = models.SaveInvitation(invitation)
		require.NoError(t, err)

		canvasIDs := []string{canvas.ID.String()}

		_, err = UpdateInvitation(ctx, r.AuthService, orgID, invitation.ID.String(), canvasIDs)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})
}
