package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test_RemoveSubject(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	canvasID := r.Canvas.ID.String()

	t.Run("user not found -> error", func(t *testing.T) {
		_, err := RemoveSubject(ctx, r.AuthService, orgID, canvasID, pbAuth.SubjectIdentifierType_USER_ID, uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("removes user with viewer role from canvas", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		_, err := AddUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)

		response, err := RemoveSubject(ctx, r.AuthService, orgID, canvasID, pbAuth.SubjectIdentifierType_USER_ID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		roles, err := r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvasID)
		require.NoError(t, err)
		require.Empty(t, roles)
	})

	t.Run("removes user with admin role from canvas", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		err := r.AuthService.AssignRole(newUser.ID.String(), models.RoleCanvasAdmin, canvasID, models.DomainTypeCanvas)
		require.NoError(t, err)
		_, err = AddUser(ctx, r.AuthService, orgID, canvasID, newUser.ID.String())
		require.NoError(t, err)

		response, err := RemoveSubject(ctx, r.AuthService, orgID, canvasID, pbAuth.SubjectIdentifierType_USER_ID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		roles, err := r.AuthService.GetUserRolesForCanvas(newUser.ID.String(), canvasID)
		require.NoError(t, err)
		require.Empty(t, roles)
	})

	t.Run("removes invitation from canvas", func(t *testing.T) {
		invitation := &models.OrganizationInvitation{
			Email:          "test@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
			CanvasIDs:      datatypes.NewJSONType([]string{r.Canvas.ID.String()}),
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		response, err := RemoveSubject(ctx, r.AuthService, orgID, canvasID, pbAuth.SubjectIdentifierType_INVITATION_ID, invitation.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		updatedInvitation, err := models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)
		assert.NotContains(t, updatedInvitation.CanvasIDs.Data(), r.Canvas.ID.String())
	})

	t.Run("invitation not associated with canvas -> error", func(t *testing.T) {
		invitation := &models.OrganizationInvitation{
			Email:          "test2@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		_, err = RemoveSubject(ctx, r.AuthService, orgID, canvasID, pbAuth.SubjectIdentifierType_INVITATION_ID, invitation.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "invitation not associated with this canvas", s.Message())
	})
}
