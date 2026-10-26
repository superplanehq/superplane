package polar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// fakeCheckoutClient answers checkout reads from memory.
type fakeCheckoutClient struct {
	checkouts map[string]CheckoutSession
	err       error
	calls     int
}

func (c *fakeCheckoutClient) GetCheckout(_ context.Context, id string) (*CheckoutSession, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	checkout, ok := c.checkouts[id]
	if !ok {
		return nil, errors.New("checkout not found")
	}
	return &checkout, nil
}

func (c *fakeCheckoutClient) ListCheckouts(context.Context, string, string) ([]CheckoutSession, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	var checkouts []CheckoutSession
	for _, checkout := range c.checkouts {
		checkouts = append(checkouts, checkout)
	}
	return checkouts, nil
}

// openTeamCheckout records an open checkout intent for three seats of a Team
// product and returns it.
func openTeamCheckout(t *testing.T, org *models.Organization, checkoutID string) *billingmodels.OrganizationBillingCheckout {
	t.Helper()
	db := database.Conn()
	intent, claimed, err := billingmodels.ClaimOrganizationBillingCheckout(db, org.ID, "team", "prod_team", new(3))
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, intent.MarkOpen(db, billingmodels.PolarCheckoutSnapshot{CheckoutID: checkoutID, URL: "https://buy.polar.sh/" + checkoutID, Status: "open"}))
	return intent
}

func unresolvedCheckout(t *testing.T, org *models.Organization) *billingmodels.OrganizationBillingCheckout {
	t.Helper()
	intent, err := billingmodels.FindUnresolvedOrganizationBillingCheckout(database.Conn(), org.ID)
	require.NoError(t, err)
	return intent
}

func TestPrepareAdmissionReleasesOnlyAVerifiedTerminalCheckout(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     CheckoutStatus
		readFails  bool
		unresolved bool
	}{
		{name: "expired", status: CheckoutStatusExpired},
		{name: "failed", status: CheckoutStatusFailed},
		{name: "still payable", status: CheckoutStatusConfirmed, unresolved: true},
		{name: "Polar cannot answer", readFails: true, unresolved: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.Conn(), org.ID, "cust_1"))
			intent := openTeamCheckout(t, org, "chk_team")
			client := &fakeCheckoutClient{checkouts: map[string]CheckoutSession{
				"chk_team": {ID: "chk_team", URL: "https://buy.polar.sh/chk_team", ProductID: "prod_team", Status: tc.status, Seats: new(3)},
			}}
			if tc.readFails {
				client.err = errors.New("polar unavailable")
			}

			require.NoError(t, NewCheckoutLifecycle(client).PrepareAdmission(t.Context(), org.ID))

			stored := unresolvedCheckout(t, org)
			if tc.unresolved {
				require.NotNil(t, stored, "the restriction stays until Polar proves the checkout cannot activate")
				assert.Equal(t, intent.ID, stored.ID)
				return
			}
			assert.Nil(t, stored)
		})
	}
}

func TestPrepareAdmissionReadsPolarOnlyForAPolarCustomer(t *testing.T) {
	org := setupBillingOrganization(t)
	client := &fakeCheckoutClient{err: errors.New("polar unavailable")}

	require.NoError(t, NewCheckoutLifecycle(client).PrepareAdmission(t.Context(), org.ID))
	assert.Zero(t, client.calls, "an organization without a Polar customer has no checkout")

	require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.Conn(), org.ID, "cust_1"))
	err := NewCheckoutLifecycle(client).PrepareAdmission(t.Context(), org.ID)
	require.Error(t, err, "an unrecorded checkout from before intents could still activate")
}

func TestPrepareAdmissionAdoptsAPayableCheckoutFromBeforeIntents(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.Conn(), org.ID, "cust_1"))
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	client := &fakeCheckoutClient{checkouts: map[string]CheckoutSession{
		"chk_legacy": {ID: "chk_legacy", URL: "https://buy.polar.sh/chk_legacy", ProductID: "prod_business", Status: CheckoutStatusOpen, Seats: new(2)},
	}}

	require.NoError(t, NewCheckoutLifecycle(client).PrepareAdmission(t.Context(), org.ID))

	stored := unresolvedCheckout(t, org)
	require.NotNil(t, stored)
	assert.Equal(t, "chk_legacy", stored.PolarCheckoutIDValue())
	assert.Equal(t, new(2), stored.Seats)
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return CheckCheckoutAdmission(tx, org.ID, 2)
	}))
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return CheckCheckoutAdmission(tx, org.ID, 3)
	})
	require.ErrorIs(t, err, ErrCheckoutSeatLimit, "a legacy payable checkout cannot admit an unpaid member")
}

func TestCheckCheckoutAdmissionKeepsMembersWithinTheSelectedSeats(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	check := func(members int) error {
		return db.Transaction(func(tx *gorm.DB) error { return CheckCheckoutAdmission(tx, org.ID, members) })
	}
	require.NoError(t, check(10), "no checkout restricts membership")

	// A creating intent counts: its checkout can still exist at Polar.
	intent, claimed, err := billingmodels.ClaimOrganizationBillingCheckout(db, org.ID, "team", "prod_team", new(3))
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, check(3))
	require.ErrorIs(t, check(4), ErrCheckoutSeatLimit)

	require.NoError(t, intent.Resolve(db, billingmodels.BillingCheckoutStateExpired, nil))
	require.NoError(t, check(4), "a resolved checkout restricts nothing")
}

func TestActivationResolvesTheCheckoutThatPolarNames(t *testing.T) {
	org := setupBillingOrganization(t)
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	intent, claimed, err := billingmodels.ClaimOrganizationBillingCheckout(database.Conn(), org.ID, billingmodels.BillingPlanBusiness, "prod_business", nil)
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, intent.MarkOpen(database.Conn(), billingmodels.PolarCheckoutSnapshot{CheckoutID: "chk_paid", URL: "https://buy.polar.sh/chk_paid", Status: "open"}))
	now := time.Now().UTC().Truncate(time.Second)
	event := subscriptionEvent(org.ID, "sub_activation", "active", now, now.AddDate(0, 1, 0))

	// The same product from another checkout, or without a checkout, does
	// not describe this intent.
	for i, checkoutID := range []string{"chk_other", ""} {
		event.Data.CheckoutID = checkoutID
		event.Data.ModifiedAt = polarTime{Time: now.Add(time.Duration(i) * time.Second)}
		require.NoError(t, ApplySubscriptionEvent(t.Context(), database.Conn(), event))
		require.NotNil(t, unresolvedCheckout(t, org), "checkout %q must not resolve the intent", checkoutID)
	}

	event.Data.CheckoutID = "chk_paid"
	event.Data.ModifiedAt = polarTime{Time: now.Add(time.Minute)}
	require.NoError(t, ApplySubscriptionEvent(t.Context(), database.Conn(), event))

	assert.Nil(t, unresolvedCheckout(t, org))
	var resolved billingmodels.OrganizationBillingCheckout
	require.NoError(t, database.Conn().First(&resolved, "id = ?", intent.ID).Error)
	assert.Equal(t, billingmodels.BillingCheckoutStateSucceeded, resolved.State)
}
