package organizations

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/testsupport/polarfake"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

// Test__RenewalEstimateComesFromTheStoredPlan runs the billing API on the
// stored plan. The estimate keeps its current formula until the billing
// presentation work replaces it. Describing billing never writes a seat count
// to Polar.
func Test__RenewalEstimateComesFromTheStoredPlan(t *testing.T) {
	tests := []struct {
		name      string
		paidSeats int
		// members includes the owner.
		members            int
		pendingInvitations int
		// change runs after the seed, before the API.
		change func(t *testing.T, state renewalEstimateState)
		// expected is the renewal seat count. Nil means that the paid seats
		// are invalid, so there is no estimate.
		expected *int
	}{
		{name: "ordinary", paidSeats: 10, members: 3, expected: new(3)},
		{name: "no members renew one seat", paidSeats: 10, members: 1, expected: new(1), change: func(t *testing.T, state renewalEstimateState) {
			require.NoError(t, database.Conn().Delete(&models.User{}, "id = ?", state.r.User).Error)
		}},
		{name: "over capacity renews only the paid seats", paidSeats: 1, members: 3, expected: new(1)},
		{name: "pending invitations do not count", paidSeats: 10, members: 3, pendingInvitations: 2, expected: new(3)},
		{name: "plan maximum change creates no paid seats", paidSeats: 1, members: 3, expected: new(1), change: func(t *testing.T, state renewalEstimateState) {
			require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), state.r.Organization.ID, new(80)))
		}},
		{name: "zero paid seats", paidSeats: 10, members: 3, change: func(t *testing.T, state renewalEstimateState) {
			state.storeInvalidPaidSeats(t, 0)
		}},
		{name: "missing paid seats", paidSeats: 10, members: 3, change: func(t *testing.T, state renewalEstimateState) {
			state.storeInvalidPaidSeats(t, nil)
		}},
	}
	for _, payment := range renewalPaymentStates {
		for _, tt := range tests {
			t.Run(payment.name+"/"+tt.name, func(t *testing.T) {
				state := newRenewalEstimateState(t, tt.paidSeats, tt.members, tt.pendingInvitations)
				if payment.failedFor > 0 {
					state.failRenewalPayment(t, time.Now().Add(-payment.failedFor))
				}
				if tt.change != nil {
					tt.change(t, state)
				}

				billing, err := DescribeOrganizationBilling(context.Background(), state.r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
				require.NoError(t, err)
				require.Equal(t, !payment.fallback, billing.PlanPerSeat)
				assert.Empty(t, state.polar.Patches(), "describing billing never writes a seat count")

				if tt.expected == nil {
					assert.Nil(t, billing.ExpectedRenewalSeats, "invalid paid seats get no estimate")
					assert.Nil(t, billing.ExpectedRenewalCents)
					assert.Equal(t, int32(0), billing.GetPurchasedSeats(), "no paid seats are invented")
					return
				}

				require.NotNil(t, billing.ExpectedRenewalSeats)
				assert.Equal(t, int32(*tt.expected), *billing.ExpectedRenewalSeats)
				require.NotNil(t, billing.ExpectedRenewalCents)
				assert.Equal(t, int64(*tt.expected)*state.paidSeatPriceCents(t), *billing.ExpectedRenewalCents,
					"the estimate uses the paid seat price")
				if payment.fallback {
					assert.NotEqual(t, int64(*tt.expected)*billing.PlanPriceCents, *billing.ExpectedRenewalCents,
						"the estimate never uses the access plan price")
				}
				assert.Equal(t, int32(tt.paidSeats), billing.GetPurchasedSeats(), "the paid seats do not change")
				assert.Equal(t, int32(tt.pendingInvitations), billing.GetPendingInvitationSeats())
			})
		}
	}
}

// renewalEstimateGraceDays is the past-due grace period of the renewal
// estimate tests.
const renewalEstimateGraceDays = 7

// renewalPaymentStates are the payment states in which a subscription renews.
var renewalPaymentStates = []struct {
	name string
	// failedFor is how long ago the renewal payment failed. Zero keeps the
	// payment current.
	failedFor time.Duration
	// fallback is true when the failure has moved the plan row to the access
	// plan.
	fallback bool
}{
	{name: "active"},
	{name: "past due in grace", failedFor: 24 * time.Hour},
	{name: "past due after grace", failedFor: (renewalEstimateGraceDays + 1) * 24 * time.Hour, fallback: true},
}

// Test__BlockedRenewalAmountIsUnavailableWhenPolarFails checks that a Polar
// failure after the fallback hides only the amount. The billing page still
// loads, and the amount is not zero.
func Test__BlockedRenewalAmountIsUnavailableWhenPolarFails(t *testing.T) {
	state := newRenewalEstimateState(t, 5, 3, 0)
	state.failRenewalPayment(t, time.Now().Add(-(renewalEstimateGraceDays+1)*24*time.Hour))
	state.polar.FailNext(1, http.StatusTooManyRequests, "rate_limited")

	billing, err := DescribeOrganizationBilling(context.Background(), state.r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	require.False(t, billing.PlanPerSeat, "the row is on the access plan")
	require.NotNil(t, billing.ExpectedRenewalSeats)
	assert.Equal(t, int32(3), *billing.ExpectedRenewalSeats)
	assert.Nil(t, billing.ExpectedRenewalCents, "an unverified price is not available")
}

type renewalEstimateState struct {
	r     *support.ResourceRegistry
	polar *polarfake.Server
	sub   polarfake.Subscription
}

func newRenewalEstimateState(t *testing.T, paidSeats, members, pendingInvitations int) renewalEstimateState {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	state := renewalEstimateState{r: r, polar: polarfake.NewServer(t)}
	state.sub = state.polar.AddSeatPlan(database.Conn(), r.Organization.ID, paidSeats, time.Now().Add(10*24*time.Hour))
	for range members - 1 {
		support.CreateUser(t, r, r.Organization.ID)
	}
	now := time.Now()
	for i := range pendingInvitations {
		invitation := models.OrganizationInvitation{
			OrganizationID: r.Organization.ID, Email: support.RandomName("pending") + "@example.com",
			Role: models.RoleOrgOperator, InvitedBy: r.User, State: models.InvitationStatePending,
			SentAt: now.Add(time.Duration(i) * time.Second), ExpiresAt: now.Add(time.Hour),
		}
		require.NoError(t, invitation.Save(database.Conn()))
	}
	return state
}

// failRenewalPayment marks the subscription past due since the given time,
// in SuperPlane and in Polar.
func (s renewalEstimateState) failRenewalPayment(t *testing.T, since time.Time) {
	t.Helper()
	require.NoError(t, billingmodels.UpdatePastDueGraceDays(database.Conn(), renewalEstimateGraceDays))
	require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", s.r.Organization.ID).
		Updates(map[string]any{
			"polar_subscription_status": billingmodels.PolarSubscriptionStatusPastDue,
			"past_due_since":            since.UTC(),
		}).Error)
	s.polar.Update(s.sub.ID, func(sub *polarfake.Subscription) { sub.Status = billingmodels.PolarSubscriptionStatusPastDue })
}

// paidSeatPriceCents is the catalog price of the paid product of the
// subscription.
func (s renewalEstimateState) paidSeatPriceCents(t *testing.T) int64 {
	t.Helper()
	catalog, err := billingmodels.FindBillingPlanByProduct(database.Conn(), s.sub.ProductID)
	require.NoError(t, err)
	return catalog.PriceCents
}

// storeInvalidPaidSeats stores paid seats that need repair. Polar answers
// with an older snapshot, so a refresh keeps the stored value.
func (s renewalEstimateState) storeInvalidPaidSeats(t *testing.T, seats any) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", s.r.Organization.ID).
		Updates(map[string]any{"purchased_seats": seats, "polar_modified_at": now}).Error)
	s.polar.Update(s.sub.ID, func(sub *polarfake.Subscription) { sub.ModifiedAt = new(now.Add(-time.Hour)) })
}
