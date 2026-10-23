package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestSharedInviteLinkRequiresEmailInvitation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		reservedSeat bool
		removed      bool
	}{
		{name: "reserved paid seat", reservedSeat: true},
		{name: "billing disabled"},
		{name: "removed member", removed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "no")
			db := database.Conn()
			if tc.reservedSeat {
				t.Setenv(models.BillingEnabledEnv, "yes")
				setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
				invitations := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
				_, err := invitations.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "intended@example.com"})
				require.NoError(t, err)
			}
			account, err := models.CreateAccount("Uninvited person", uuid.NewString()+"@example.com")
			require.NoError(t, err)
			if tc.removed {
				user, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
				require.NoError(t, err)
				require.NoError(t, db.Delete(user).Error)
			}
			before, err := entitlements.ResolveSeats(db, r.Organization.ID)
			require.NoError(t, err)
			if tc.reservedSeat {
				require.Equal(t, 2, *before.Purchased)
				require.Equal(t, 1, before.Occupied)
				require.Equal(t, 1, before.PendingInvitations)
				require.Zero(t, *before.Invitable)
			}
			link, err := models.FindOrCreateInviteLink(db, r.Organization.ID)
			require.NoError(t, err)

			response, err := AcceptInviteLink(context.Background(), r.AuthService, account.ID.String(), link.Token.String())

			require.Nil(t, response)
			code, message, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok)
			require.Equal(t, codes.FailedPrecondition, code)
			require.Equal(t, "This organization uses email invitations. Ask an organization admin for an invitation.", message)
			_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
			after, err := entitlements.ResolveSeats(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}

func TestSharedInviteLinkRedirectsExistingMember(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.Conn()
	link, err := models.FindOrCreateInviteLink(db, r.Organization.ID)
	require.NoError(t, err)

	response, err := AcceptInviteLink(context.Background(), r.AuthService, r.Account.ID.String(), link.Token.String())

	require.NoError(t, err)
	require.Equal(t, "already_member", response.Fields["status"].GetStringValue())
	require.Equal(t, r.Organization.ID.String(), response.Fields["organization_id"].GetStringValue())
	require.Equal(t, r.Organization.Slug, response.Fields["organization_slug"].GetStringValue())
	occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
	require.NoError(t, err)
	require.EqualValues(t, 1, occupied)
}

func seatQuantity(quantity int) *int { return &quantity }

func setPurchasedSeatPlan(t *testing.T, db *gorm.DB, orgID uuid.UUID, purchased *int) string {
	t.Helper()
	maximum := 50
	product := "product_" + uuid.NewString()
	plan := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: &maximum, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).Updates(map[string]any{"plan_id": plan.ID, "plan": plan.Key, "plan_source": "polar", "polar_subscription_status": "active", "purchased_seats": purchased}).Error)
	return product
}
