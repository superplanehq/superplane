package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test__AcceptInviteLinkWithUsage(t *testing.T) {
	r := support.Setup(t)

	t.Run("usage limit violation blocks joining organization", func(t *testing.T) {
		account, err := models.CreateAccount(support.RandomName("account")+"@example.com", support.RandomName("user"))
		require.NoError(t, err)
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)
		userCount, err := models.CountActiveHumanUsersByOrganization(r.Organization.ID.String())
		require.NoError(t, err)

		service := &fakeUsageService{
			enabled: true,
			checkOrganizationResp: &usagepb.CheckOrganizationLimitsResponse{
				Allowed: false,
				Violations: []*usagepb.LimitViolation{
					{
						Limit:           usagepb.LimitName_LIMIT_NAME_MAX_USERS,
						ConfiguredLimit: 1,
						CurrentValue:    2,
					},
				},
			},
		}

		_, err = AcceptInviteLinkWithUsage(context.Background(), r.AuthService, service, account.ID.String(), inviteLink.Token.String())
		require.Error(t, err)
		assert.Equal(t, codes.ResourceExhausted, grpcerrors.Code(err))
		assert.Equal(t, "organization user limit exceeded", status.Convert(err).Message())
		require.Len(t, service.checkOrganizationCalls, 1)
		assert.Equal(t, int32(userCount+1), service.checkOrganizationCalls[0].state.Users)

		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("already member bypasses usage check", func(t *testing.T) {
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)

		service := &fakeUsageService{
			enabled: true,
			checkOrganizationResp: &usagepb.CheckOrganizationLimitsResponse{
				Allowed: false,
				Violations: []*usagepb.LimitViolation{
					{
						Limit:           usagepb.LimitName_LIMIT_NAME_MAX_USERS,
						ConfiguredLimit: 1,
						CurrentValue:    2,
					},
				},
			},
		}

		response, err := AcceptInviteLinkWithUsage(
			context.Background(),
			r.AuthService,
			service,
			r.Account.ID.String(),
			inviteLink.Token.String(),
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Empty(t, service.checkOrganizationCalls)
		assert.Equal(t, "already_member", response.Fields["status"].GetStringValue())
	})
}

func TestNotifyOrganizationOwnersOfJoinedMember(t *testing.T) {
	r := support.Setup(t)
	memberEmail := "alex@example.com"
	member := &models.User{Name: "Alex", Email: &memberEmail}
	var published []messages.OrganizationMemberJoinedMessage

	notifyOrganizationOwnersOfJoinedMember(
		context.Background(),
		r.AuthService,
		func(message messages.OrganizationMemberJoinedMessage) error {
			published = append(published, message)
			return nil
		},
		func(_ *gorm.DB, organizationID string, ownerIDs []string) ([]models.User, error) {
			require.Equal(t, r.Organization.ID.String(), organizationID)
			require.Contains(t, ownerIDs, r.UserModel.ID.String())
			return []models.User{*r.UserModel}, nil
		},
		r.Organization,
		member,
	)

	require.Len(t, published, 1)
	assert.Equal(t, r.UserModel.GetEmail(), published[0].ToEmail)
	assert.Equal(t, "Alex", published[0].MemberName)
	assert.Equal(t, "alex@example.com", published[0].MemberEmail)
}

func TestAcceptInviteLinkPublishesOrganizationMemberJoinedNotifications(t *testing.T) {
	t.Run("publishes one message for each eligible owner after a new member joins", func(t *testing.T) {
		r := support.Setup(t)
		secondOwnerAccount, err := models.CreateAccount("second-owner@example.com", "Second Owner")
		require.NoError(t, err)
		secondOwner, err := models.CreateUser(r.Organization.ID, secondOwnerAccount.ID, secondOwnerAccount.Email, secondOwnerAccount.Name)
		require.NoError(t, err)
		require.NoError(t, r.AuthService.AssignRole(secondOwner.ID.String(), models.RoleOrgOwner, r.Organization.ID.String(), models.DomainTypeOrganization))
		invitee, err := models.CreateAccount("invitee@example.com", "Invitee")
		require.NoError(t, err)
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)

		var published []messages.OrganizationMemberJoinedMessage
		response, err := acceptInviteLinkWithUsage(
			context.Background(),
			r.AuthService,
			nil,
			func(message messages.OrganizationMemberJoinedMessage) error {
				published = append(published, message)
				return nil
			},
			models.ListActiveHumanUsersByIDs,
			invitee.ID.String(),
			inviteLink.Token.String(),
		)
		require.NoError(t, err)
		assert.Equal(t, "joined", response.Fields["status"].GetStringValue())
		require.Len(t, published, 2)
		assert.ElementsMatch(t, []string{r.UserModel.GetEmail(), secondOwner.GetEmail()}, []string{published[0].ToEmail, published[1].ToEmail})
		for _, message := range published {
			assert.Equal(t, r.Organization.ID.String(), message.OrganizationID)
			assert.Equal(t, r.Organization.Name, message.OrganizationName)
			assert.Equal(t, invitee.Email, message.MemberEmail)
			assert.Equal(t, invitee.Name, message.MemberName)
		}

		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), invitee.Email)
		require.NoError(t, err)
	})

	t.Run("publishes for a restored member", func(t *testing.T) {
		r := support.Setup(t)
		invitee, err := models.CreateAccount("restored@example.com", "Restored")
		require.NoError(t, err)
		deletedUser, err := models.CreateUser(r.Organization.ID, invitee.ID, invitee.Email, invitee.Name)
		require.NoError(t, err)
		require.NoError(t, deletedUser.Delete())
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)

		var published []messages.OrganizationMemberJoinedMessage
		response, err := acceptInviteLinkWithUsage(
			context.Background(),
			r.AuthService,
			nil,
			func(message messages.OrganizationMemberJoinedMessage) error {
				published = append(published, message)
				return nil
			},
			models.ListActiveHumanUsersByIDs,
			invitee.ID.String(),
			inviteLink.Token.String(),
		)
		require.NoError(t, err)
		assert.Equal(t, "joined", response.Fields["status"].GetStringValue())
		require.Len(t, published, 1)
		assert.Equal(t, r.UserModel.GetEmail(), published[0].ToEmail)

		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), invitee.Email)
		require.NoError(t, err)
	})

	t.Run("does not publish for an already active member", func(t *testing.T) {
		r := support.Setup(t)
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)

		response, err := acceptInviteLinkWithUsage(
			context.Background(),
			r.AuthService,
			nil,
			func(messages.OrganizationMemberJoinedMessage) error {
				t.Fatal("already-active membership must not publish a notification")
				return nil
			},
			func(*gorm.DB, string, []string) ([]models.User, error) {
				t.Fatal("already-active membership must not resolve owners")
				return nil, nil
			},
			r.Account.ID.String(),
			inviteLink.Token.String(),
		)
		require.NoError(t, err)
		assert.Equal(t, "already_member", response.Fields["status"].GetStringValue())
	})

	t.Run("keeps the membership when publication fails", func(t *testing.T) {
		r := support.Setup(t)
		invitee, err := models.CreateAccount("publisher-error@example.com", "Publisher Error")
		require.NoError(t, err)
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)

		response, err := acceptInviteLinkWithUsage(
			context.Background(),
			r.AuthService,
			nil,
			func(messages.OrganizationMemberJoinedMessage) error {
				return errors.New("rabbitmq unavailable")
			},
			models.ListActiveHumanUsersByIDs,
			invitee.ID.String(),
			inviteLink.Token.String(),
		)
		require.NoError(t, err)
		assert.Equal(t, "joined", response.Fields["status"].GetStringValue())

		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), invitee.Email)
		require.NoError(t, err)
	})

	t.Run("keeps the membership when owner lookup fails", func(t *testing.T) {
		r := support.Setup(t)
		invitee, err := models.CreateAccount("owner-lookup-error@example.com", "Owner Lookup Error")
		require.NoError(t, err)
		inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(t.Context()), r.Organization.ID.String())
		require.NoError(t, err)

		response, err := acceptInviteLinkWithUsage(
			context.Background(),
			r.AuthService,
			nil,
			func(messages.OrganizationMemberJoinedMessage) error {
				t.Fatal("owner lookup failure must not publish a notification")
				return nil
			},
			func(*gorm.DB, string, []string) ([]models.User, error) {
				return nil, errors.New("database unavailable")
			},
			invitee.ID.String(),
			inviteLink.Token.String(),
		)
		require.NoError(t, err)
		assert.Equal(t, "joined", response.Fields["status"].GetStringValue())

		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), invitee.Email)
		require.NoError(t, err)
	})
}
