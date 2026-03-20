package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		inviteLink, err := models.FindInviteLinkByOrganizationID(r.Organization.ID.String())
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
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.ResourceExhausted, s.Code())
		assert.Equal(t, "organization user limit exceeded", s.Message())
		require.Len(t, service.checkOrganizationCalls, 1)
		assert.Equal(t, int32(userCount+1), service.checkOrganizationCalls[0].state.Users)

		_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("already member bypasses usage check", func(t *testing.T) {
		inviteLink, err := models.FindInviteLinkByOrganizationID(r.Organization.ID.String())
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
