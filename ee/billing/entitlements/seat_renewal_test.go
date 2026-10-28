package entitlements_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
)

func TestRenewalSeatsKeepsOccupiedSeatsWithinPaidSeats(t *testing.T) {
	perSeat := billingmodels.BillingPlan{PerSeat: true}
	tests := []struct {
		name     string
		plan     billingmodels.OrganizationBillingPlan
		occupied int
		expected *int
		err      error
	}{
		{name: "members below paid seats", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat, PurchasedSeats: seatCount(10)}, occupied: 9, expected: seatCount(9)},
		{name: "members equal paid seats", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat, PurchasedSeats: seatCount(10)}, occupied: 10, expected: seatCount(10)},
		{name: "members above paid seats never add seats", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat, PurchasedSeats: seatCount(10)}, occupied: 12, expected: seatCount(10)},
		{name: "three members and one paid seat renew one seat", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat, PurchasedSeats: seatCount(1)}, occupied: 3, expected: seatCount(1)},
		{name: "no members keep one seat", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat, PurchasedSeats: seatCount(10)}, occupied: 0, expected: seatCount(1)},
		{name: "fallback catalog keeps the paid seats", plan: billingmodels.OrganizationBillingPlan{PurchasedSeats: seatCount(5)}, occupied: 3, expected: seatCount(3)},
		{name: "missing paid seats", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat}, occupied: 3, err: entitlements.ErrInvalidPaidSeats},
		{name: "zero paid seats", plan: billingmodels.OrganizationBillingPlan{Catalog: perSeat, PurchasedSeats: seatCount(0)}, occupied: 3, err: entitlements.ErrInvalidPaidSeats},
		{name: "not seat-based", plan: billingmodels.OrganizationBillingPlan{}, occupied: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := entitlements.RenewalSeats(&tt.plan, tt.occupied)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestPastDueLapseKeepsRenewalSeatsButLocksAdmission(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	support.CreateUser(t, r, r.Organization.ID)
	replay := newSubscriptionReplay(t, r.Organization.ID, savePaidCatalog(t, true))
	replay.apply("active", seatCount(5))

	// No grace period applies, so the failure moves the row to the fallback
	// catalog at once.
	replay.apply("past_due", seatCount(5))

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.False(t, plan.Catalog.PerSeat, "the row is on the fallback catalog")
	usage, err := entitlements.ResolveSeatsForPlan(db, r.Organization.ID, plan)
	require.NoError(t, err)
	require.True(t, usage.PaymentBlocked)
	require.Equal(t, seatCount(2), usage.RenewalQuantity, "the seat summary shows the renewal estimate while payment is blocked")
	require.Equal(t, seatCount(0), usage.Invitable)
	require.Equal(t, seatCount(0), usage.Purchasable)
	require.Equal(t, seatCount(2), entitlements.JoinLimit(usage))
}
