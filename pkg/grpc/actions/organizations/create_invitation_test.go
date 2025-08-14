package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test__CreateInvitation(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := CreateInvitation(context.Background(), r.AuthService, r.Organization.ID.String(), "new@example.com")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Equal(t, "user not authenticated", s.Message())
	})

	t.Run("empty email -> error", func(t *testing.T) {
		_, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "email is required", s.Message())
	})

	t.Run("user already exists in organization -> error", func(t *testing.T) {
		_, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), r.Account.Email)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
		assert.Equal(t, "user is already a member of the organization", s.Message())
	})

	t.Run("account does not exist -> creates pending invitation", func(t *testing.T) {
		email := "does-not-exist@example.com"
		response, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), email)
		require.NoError(t, err)
		assert.Equal(t, r.Organization.ID.String(), response.Invitation.OrganizationId)
		assert.Equal(t, email, response.Invitation.Email)
		assert.Equal(t, models.InvitationStatusPending, response.Invitation.Status)

		// Verify user for this account is not added to organization
		_, err = models.FindUserByEmail(r.Organization.ID.String(), email)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("account exists -> creates accepted invitation and adds user immediately", func(t *testing.T) {
		//
		// Create a separate account that is not yet in the organization
		//
		account, err := models.CreateAccount("existing@example.com", "Existing User")
		require.NoError(t, err)

		response, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		assert.Equal(t, r.Organization.ID.String(), response.Invitation.OrganizationId)
		assert.Equal(t, account.Email, response.Invitation.Email)
		assert.Equal(t, models.InvitationStatusAccepted, response.Invitation.Status)

		//
		// Verify the user was created in the organization and assigned the viewer role
		//
		user, err := models.FindUserByEmail(r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		assert.Equal(t, account.ID, user.AccountID)
		assert.Equal(t, account.Email, user.Email)
		assert.Equal(t, account.Name, user.Name)
		assert.Equal(t, r.Organization.ID, user.OrganizationID)

		roles, err := r.AuthService.GetUserRolesForOrg(user.ID.String(), r.Organization.ID.String())
		require.NoError(t, err)
		assert.Contains(t, roles[0].Name, models.RoleOrgViewer)
	})

	t.Run("duplicate invitation for non-existent account -> error", func(t *testing.T) {
		email := "duplicate@example.com"

		// Create first invitation
		_, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), email)
		require.NoError(t, err)

		// Try to create second invitation for same email
		_, err = CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), email)
		assert.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "Failed to create invitation")
	})
}
