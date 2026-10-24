package organizations

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

// replay delivers a Polar webhook with the given status, newer than the last one.
func (f *seatPurchaseFixture) replay(status string, seats int) {
	f.t.Helper()
	f.replays++
	data := f.snapshot(seats)
	data["status"] = status
	data["modified_at"] = time.Now().Add(time.Duration(f.replays) * time.Minute).UTC().Format(time.RFC3339)
	body, err := json.Marshal(map[string]any{"type": "subscription.updated", "data": data})
	require.NoError(f.t, err)
	var event polar.SubscriptionWebhookEvent
	require.NoError(f.t, json.Unmarshal(body, &event))
	require.NoError(f.t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
}

// saveBusinessPlan adds a public paid plan without seats, like Business.
func saveBusinessPlan(t *testing.T) *models.BillingPlan {
	t.Helper()
	product := "prod_" + uuid.NewString()
	catalog := models.BillingPlan{Key: "business_" + uuid.NewString()[:8], Name: "Business", Category: "paid", Visibility: "public", PriceCents: 8000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(database.Conn(), &catalog))
	return &catalog
}

// saveFreePlan adds a public paid plan that needs no checkout and no card.
func saveFreePlan(t *testing.T) *models.BillingPlan {
	t.Helper()
	catalog := models.BillingPlan{Key: "free_" + uuid.NewString()[:8], Name: "Free", Category: "paid", Visibility: "public", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(database.Conn(), &catalog))
	return &catalog
}

func TestPastDueRenewalLocksMembershipUntilPaymentRecovers(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	db := database.Conn()
	orgID := f.r.Organization.ID
	support.CreateUser(t, f.r, orgID)
	service := InvitationService{AuthService: f.r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
	create := func(email string) (*pb.CreateInvitationResponse, error) {
		return service.Create(context.Background(), orgID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: email})
	}
	occupied := func() int64 {
		count, err := models.CountActiveHumanUsersByOrganization(db, orgID.String())
		require.NoError(t, err)
		return count
	}

	invitee, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	pending, err := create(invitee.Email)
	require.NoError(t, err)
	token := strings.TrimPrefix(pending.AcceptanceUrl, "https://example.com/invitations/")
	require.EqualValues(t, 2, occupied())

	f.replay(models.PolarSubscriptionStatusPastDue, 3)

	_, err = create("late@example.com")
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
	_, err = service.Resend(context.Background(), orgID.String(), f.r.User.String(), pending.Invitation.Id)
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
	_, err = service.Accept(context.Background(), invitee.ID.String(), token)
	requireGRPCCode(t, codes.FailedPrecondition, err)
	require.EqualValues(t, 2, occupied())
	stored, err := models.FindOrganizationInvitation(db, orgID, uuid.MustParse(pending.Invitation.Id))
	require.NoError(t, err)
	require.Equal(t, models.InvitationStatePending, stored.State)
	listed, err := service.List(context.Background(), orgID.String())
	require.NoError(t, err)
	require.EqualValues(t, 0, *listed.Seats.Invitable)
	require.EqualValues(t, 0, *listed.Seats.Purchasable)

	require.NoError(t, db.Model(&models.BillingPlan{}).Where("id = ?", f.catalog.ID).Update("visibility", "public").Error)
	for name, request := range map[string]*pb.SelectBillingPlanRequest{
		"Team":     {PlanKey: f.catalog.Key, Seats: new(int32(3))},
		"Business": {PlanKey: saveBusinessPlan(t).Key},
		"Free":     {PlanKey: saveFreePlan(t).Key},
		"None":     {PlanKey: models.BillingPlanNone},
	} {
		_, err = SelectBillingPlan(context.Background(), orgID.String(), request, f.r.Account.ID.String(), "http://localhost")
		requireGRPCCode(t, codes.FailedPrecondition, err)
		require.Contains(t, err.Error(), "The last renewal payment failed.", name)
	}

	// A blocked selection must not clear the stored subscription state.
	locked, err := models.FindOrganizationBillingPlan(db, orgID)
	require.NoError(t, err)
	require.Equal(t, models.PolarSubscriptionStatusPastDue, locked.PolarSubscriptionStatus)
	require.Equal(t, models.BillingPlanSourcePolar, locked.PlanSource)

	f.replay(models.PolarSubscriptionStatusActive, 3)

	_, err = create("late@example.com")
	requireInvitationReason(t, err, "SEAT_PURCHASE_REQUIRED")
	_, err = service.Accept(context.Background(), invitee.ID.String(), token)
	require.NoError(t, err)
	require.EqualValues(t, 3, occupied())
}
