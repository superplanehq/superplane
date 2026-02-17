package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
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

	t.Run("active user already exists in organization -> error", func(t *testing.T) {
		_, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), r.Account.Email)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "user "+r.Account.Email+" is already an active member of organization", s.Message())
	})

	t.Run("user is added back after being removed previously", func(t *testing.T) {
		//
		// First, add user to organization
		//
		account, err := models.CreateAccount("existing@example.com", "Existing User")
		require.NoError(t, err)
		_, err = CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		user, err := models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
		require.NoError(t, err)

		//
		// Then, remove the user from the organization, and verify the user is soft-deleted.
		//
		_, err = RemoveUser(ctx, r.AuthService, r.Organization.ID.String(), user.ID.String())
		require.NoError(t, err)
		user, err = models.FindMaybeDeletedUserByID(r.Organization.ID.String(), user.ID.String())
		require.NoError(t, err)
		require.True(t, user.DeletedAt.Valid)

		//
		// Add user back to the organization,
		// and verify the user is active again.
		//
		_, err = CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		user, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		require.False(t, user.DeletedAt.Valid)

		//
		// Verify the user has 2 accepted invitations
		//
		invitations, err := models.ListInvitationsInState(r.Organization.ID.String(), models.InvitationStatePending)
		require.NoError(t, err)
		require.Len(t, invitations, 0)
		invitations, err = models.ListInvitationsInState(r.Organization.ID.String(), models.InvitationStateAccepted)
		require.NoError(t, err)
		require.Len(t, invitations, 2)
	})

	t.Run("account does not exist -> creates pending invitation", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, messages.InvitationCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		email := "does-not-exist@example.com"
		response, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), email)
		require.NoError(t, err)
		assert.Equal(t, r.Organization.ID.String(), response.Invitation.OrganizationId)
		assert.Equal(t, email, response.Invitation.Email)
		assert.Equal(t, models.InvitationStatePending, response.Invitation.State)

		// Verify user for this account is not added to organization
		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), email)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("account exists -> creates accepted invitation and adds user immediately", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, messages.InvitationCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Create a separate account that is not yet in the organization
		//
		account, err := models.CreateAccount(support.RandomName("account")+"@example.com", support.RandomName("user"))
		require.NoError(t, err)

		response, err := CreateInvitation(ctx, r.AuthService, r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		assert.Equal(t, r.Organization.ID.String(), response.Invitation.OrganizationId)
		assert.Equal(t, account.Email, response.Invitation.Email)
		assert.Equal(t, models.InvitationStateAccepted, response.Invitation.State)

		//
		// Verify the user was created in the organization and assigned the viewer role
		//
		user, err := models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
		require.NoError(t, err)
		assert.Equal(t, &account.ID, user.AccountID)
		assert.Equal(t, account.Email, user.GetEmail())
		assert.Equal(t, account.Name, user.Name)
		assert.Equal(t, r.Organization.ID, user.OrganizationID)

		roles, err := r.AuthService.GetUserRolesForOrg(user.ID.String(), r.Organization.ID.String())
		require.NoError(t, err)
		require.NotEmpty(t, roles)

		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}
		assert.Contains(t, roleNames, models.RoleOrgViewer)

		assert.False(t, testconsumer.HasReceivedMessage())
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

type mockAuthService struct {
	authorization.Authorization
	Error error
}

func (m *mockAuthService) AssignRole(userID, role, domainID string, domainType string) error {
	if m.Error != nil {
		return m.Error
	}
	return m.Authorization.AssignRole(userID, role, domainID, domainType)
}

func (m *mockAuthService) DestroyOrganization(tx *gorm.DB, orgID string) error {
	if m.Error != nil {
		return m.Error
	}
	return m.Authorization.DestroyOrganization(tx, orgID)
}

func Test__CreateInvitation_TransactionRollback(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("auth service failure rolls back invitation and user creation", func(t *testing.T) {
		account, err := models.CreateAccount("invited@example.com", "Invited User")
		require.NoError(t, err)

		//
		// Use an authentication service that fails,
		//
		mockAuth := &mockAuthService{
			Authorization: r.AuthService,
			Error:         errors.New("oops"),
		}

		//
		// Try to create invitation
		// It should fail due to role assignment error.
		//
		_, err = CreateInvitation(ctx, mockAuth, r.Organization.ID.String(), account.Email)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "oops")

		//
		// Verify that no invitation was created
		//
		invitations, err := models.ListInvitationsInState(r.Organization.ID.String(), models.InvitationStateAccepted)
		require.NoError(t, err)
		for _, inv := range invitations {
			assert.NotEqual(t, account.Email, inv.Email, "Invitation should not exist after transaction rollback")
		}

		//
		// Verify that no user was created for this account
		//
		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "User should not exist after transaction rollback")

		//
		// Verify the account still exists
		//
		foundAccount, err := models.FindAccountByEmail(account.Email)
		require.NoError(t, err)
		assert.Equal(t, account.ID, foundAccount.ID)
	})
}
