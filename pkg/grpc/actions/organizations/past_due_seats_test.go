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
	"google.golang.org/grpc/status"
)

// replay delivers a Polar webhook with the given status, newer than the last one.
func (f *seatPurchaseFixture) replay(status string, seats int) {
	f.t.Helper()
	f.replays++
	data := f.snapshot(seats)
	data["status"] = status
	if f.pastDueAt != nil {
		data["past_due_at"] = f.pastDueAt.UTC().Format(time.RFC3339)
	}
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
	f.replay(models.PolarSubscriptionStatusActive, 3)
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
	listed, err := service.List(context.Background(), orgID.String())
	require.NoError(t, err)
	require.False(t, listed.Seats.PaymentBlocked)

	f.replay(models.PolarSubscriptionStatusPastDue, 3)

	_, err = create("late@example.com")
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
	_, err = service.Resend(context.Background(), orgID.String(), f.r.User.String(), pending.Invitation.Id)
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
	_, err = service.Accept(context.Background(), invitee.ID.String(), token)
	requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
	require.Equal(t, "This organization cannot add members until its payment is updated. Ask an organization admin.", status.Convert(err).Message())
	require.EqualValues(t, 2, occupied())
	stored, err := models.FindOrganizationInvitation(db, orgID, uuid.MustParse(pending.Invitation.Id))
	require.NoError(t, err)
	require.Equal(t, models.InvitationStatePending, stored.State)
	listed, err = service.List(context.Background(), orgID.String())
	require.NoError(t, err)
	require.True(t, listed.Seats.PaymentBlocked)
	require.EqualValues(t, 0, *listed.Seats.Invitable)
	require.EqualValues(t, 0, *listed.Seats.Purchasable)
	billing, err := DescribeOrganizationBilling(context.Background(), orgID.String(), &pb.DescribeOrganizationBillingRequest{Id: orgID.String()})
	require.NoError(t, err)
	require.Equal(t, models.PolarSubscriptionStatusPastDue, billing.PolarSubscriptionStatus)
	require.NotNil(t, billing.OccupiedSeats)
	require.EqualValues(t, 2, *billing.OccupiedSeats)

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
	listed, err = service.List(context.Background(), orgID.String())
	require.NoError(t, err)
	require.False(t, listed.Seats.PaymentBlocked)

	_, err = create("late@example.com")
	requireInvitationReason(t, err, "SEAT_PURCHASE_REQUIRED")
	_, err = service.Accept(context.Background(), invitee.ID.String(), token)
	require.NoError(t, err)
	require.EqualValues(t, 3, occupied())
}

func TestPastDueGraceKeepsTeamOnButAdmitsNoMembers(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	db := database.Conn()
	orgID := f.r.Organization.ID
	support.CreateUser(t, f.r, orgID)
	require.NoError(t, models.ApplyInstallationMetadata(db, func(metadata *models.InstallationMetadata) {
		metadata.PastDueGraceDays = 7
	}))
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).
		Update("trial_ends_at", time.Now().Add(-time.Hour)).Error)
	service := InvitationService{AuthService: f.r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
	invitee, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	pending, err := service.Create(context.Background(), orgID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: invitee.Email})
	require.NoError(t, err)
	token := strings.TrimPrefix(pending.AcceptanceUrl, "https://example.com/invitations/")
	describe := func() *pb.DescribeOrganizationBillingResponse {
		response, err := DescribeOrganizationBilling(context.Background(), orgID.String(), &pb.DescribeOrganizationBillingRequest{})
		require.NoError(t, err)
		return response
	}
	requireNobodyJoins := func() {
		_, err := service.Create(context.Background(), orgID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: "late@example.com"})
		requireInvitationReason(t, err, "PAYMENT_PAST_DUE")
		_, err = service.Accept(context.Background(), invitee.ID.String(), token)
		requireGRPCCode(t, codes.FailedPrecondition, err)
	}

	failure := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	f.pastDueAt = &failure
	f.replay(models.PolarSubscriptionStatusPastDue, 3)

	inGrace := describe()
	require.True(t, inGrace.HostedExecutionAllowed, "Team features stay on during the grace period")
	require.True(t, inGrace.PlanPerSeat)
	require.True(t, inGrace.PaymentFailedAt.AsTime().Equal(failure))
	require.True(t, inGrace.PaidAccessEndsAt.AsTime().Equal(failure.Add(7*24*time.Hour)))
	require.EqualValues(t, 3, *inGrace.PurchasedSeats)
	require.EqualValues(t, 2, *inGrace.OccupiedSeats)
	requireNobodyJoins()

	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).
		Update("past_due_since", time.Now().Add(-8*24*time.Hour)).Error)

	afterGrace := describe()
	require.False(t, afterGrace.HostedExecutionAllowed)
	require.False(t, afterGrace.PlanPerSeat)
	require.NotNil(t, afterGrace.PaymentFailedAt)
	require.Nil(t, afterGrace.PaidAccessEndsAt, "no access end date after the grace period")
	require.EqualValues(t, 3, *afterGrace.PurchasedSeats, "the seats summary stays while payment is blocked")
	require.EqualValues(t, 2, *afterGrace.OccupiedSeats)
	requireNobodyJoins()

	f.replay(models.PolarSubscriptionStatusActive, 3)
	recovered := describe()
	require.True(t, recovered.HostedExecutionAllowed)
	require.Nil(t, recovered.PaymentFailedAt)
	require.Nil(t, recovered.PaidAccessEndsAt)
}
