package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_AssignRole(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("user is not part of organization -> error", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, pbAuth.SubjectIdentifierType_USER_ID, uuid.NewString(), r.AuthService)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("assign role with user ID", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		resp, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, pbAuth.SubjectIdentifierType_USER_ID, newUser.ID.String(), r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("assign role with user email", func(t *testing.T) {
		newUser := support.CreateUser(t, r, r.Organization.ID)
		resp, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, pbAuth.SubjectIdentifierType_USER_EMAIL, newUser.Email, r.AuthService)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("invalid request - missing role", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, "", pbAuth.SubjectIdentifierType_USER_ID, r.User.String(), r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid role", s.Message())
	})

	t.Run("invalid request - missing user identifier", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, pbAuth.SubjectIdentifierType_USER_ID, "", r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("invalid request - invalid user ID", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, pbAuth.SubjectIdentifierType_USER_ID, "invalid-uuid", r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user not found", s.Message())
	})

	t.Run("assign canvas viewer role to invitation", func(t *testing.T) {
		canvas := support.CreateCanvas(t, r, r.Organization.ID, r.User)
		invitation := &models.OrganizationInvitation{
			Email:          "test@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		_, err = AssignRole(ctx, orgID, models.DomainTypeCanvas, canvas.ID.String(), models.RoleCanvasViewer, pbAuth.SubjectIdentifierType_INVITATION_ID, invitation.ID.String(), r.AuthService)
		require.NoError(t, err)

		updatedInvitation, err := models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)
		assert.Contains(t, updatedInvitation.CanvasIDs, canvas.ID)
	})

	t.Run("assign canvas viewer role to invitation - already assigned", func(t *testing.T) {
		canvas := support.CreateCanvas(t, r, r.Organization.ID, r.User)
		invitation := &models.OrganizationInvitation{
			Email:          "test2@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
			CanvasIDs:      []uuid.UUID{canvas.ID},
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		_, err = AssignRole(ctx, orgID, models.DomainTypeCanvas, canvas.ID.String(), models.RoleCanvasViewer, pbAuth.SubjectIdentifierType_INVITATION_ID, invitation.ID.String(), r.AuthService)
		require.NoError(t, err)

		updatedInvitation, err := models.FindInvitationByIDWithState(invitation.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)
		count := 0
		for _, canvasID := range updatedInvitation.CanvasIDs {
			if canvasID == canvas.ID {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("assign non-canvas role to invitation - error", func(t *testing.T) {
		invitation := &models.OrganizationInvitation{
			Email:          "test3@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		_, err = AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, pbAuth.SubjectIdentifierType_INVITATION_ID, invitation.ID.String(), r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "only canvas roles can be assigned to invitations", s.Message())
	})

	t.Run("assign non-viewer role to invitation - error", func(t *testing.T) {
		canvas := support.CreateCanvas(t, r, r.Organization.ID, r.User)
		invitation := &models.OrganizationInvitation{
			Email:          "test4@example.com",
			OrganizationID: r.Organization.ID,
			State:          models.InvitationStatePending,
		}
		err := models.SaveInvitation(invitation)
		require.NoError(t, err)

		_, err = AssignRole(ctx, orgID, models.DomainTypeCanvas, canvas.ID.String(), models.RoleCanvasAdmin, pbAuth.SubjectIdentifierType_INVITATION_ID, invitation.ID.String(), r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "only canvas viewer role can be assigned to invitations", s.Message())
	})

	t.Run("assign role to non-existent invitation - error", func(t *testing.T) {
		canvas := support.CreateCanvas(t, r, r.Organization.ID, r.User)

		_, err := AssignRole(ctx, orgID, models.DomainTypeCanvas, canvas.ID.String(), models.RoleCanvasViewer, pbAuth.SubjectIdentifierType_INVITATION_ID, uuid.NewString(), r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "invitation not found", s.Message())
	})

	t.Run("invalid subject identifier type - error", func(t *testing.T) {
		_, err := AssignRole(ctx, orgID, models.DomainTypeOrganization, orgID, models.RoleOrgAdmin, 999, "test", r.AuthService)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid subject identifier type", s.Message())
	})
}
