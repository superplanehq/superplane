package models_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func claimBillingCheckout(t *testing.T, orgID uuid.UUID, productID string, seats *int) *models.OrganizationBillingCheckout {
	t.Helper()
	checkout, claimed, err := models.ClaimOrganizationBillingCheckout(database.Conn(), orgID, "team", productID, seats)
	require.NoError(t, err)
	require.True(t, claimed)
	return checkout
}

func Test__ConcurrentClaimsCreateOneUnresolvedBillingCheckout(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID
	const claimers = 6

	var wg sync.WaitGroup
	start := make(chan struct{})
	ids := make([]uuid.UUID, claimers)
	created := make([]bool, claimers)
	errs := make([]error, claimers)
	for i := range claimers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = database.Conn().Transaction(func(tx *gorm.DB) error {
				checkout, claimed, err := models.ClaimOrganizationBillingCheckout(tx, orgID, "team", "prod_team", new(i+1))
				if err != nil {
					return err
				}
				ids[i], created[i] = checkout.ID, claimed
				// Keep the transaction open, so a claim without the lock overlaps.
				time.Sleep(50 * time.Millisecond)
				return nil
			})
		}()
	}
	close(start)
	wg.Wait()

	createdCount := 0
	for i := range claimers {
		require.NoError(t, errs[i])
		assert.Equal(t, ids[0], ids[i], "every claim must return the same intent")
		if created[i] {
			createdCount++
		}
	}
	assert.Equal(t, 1, createdCount)

	var rows int64
	require.NoError(t, database.Conn().Model(&models.OrganizationBillingCheckout{}).
		Where("organization_id = ?", orgID).Count(&rows).Error)
	assert.EqualValues(t, 1, rows)
	intent, err := models.FindUnresolvedOrganizationBillingCheckout(database.Conn(), orgID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingCheckoutStateCreating, intent.State)
	assert.Nil(t, intent.PolarCheckoutID, "a claim never invents a Polar identifier")
}

func Test__UnresolvedBillingCheckoutIndexRejectsASecondRow(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	first := claimBillingCheckout(t, r.Organization.ID, "prod_team", new(3))

	second := models.OrganizationBillingCheckout{
		OrganizationID: r.Organization.ID, PlanKey: "team", PolarProductID: "prod_team", State: models.BillingCheckoutStateOpen,
	}
	require.Error(t, db.Create(&second).Error)

	require.NoError(t, first.Release(db, "rejected"))
	second.ID = uuid.Nil
	require.NoError(t, db.Create(&second).Error, "a resolved intent no longer blocks a new one")
}

func Test__BillingCheckoutTransitionsRejectStaleWriters(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	intent := claimBillingCheckout(t, r.Organization.ID, "prod_team", new(3))
	stale := *intent

	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	snapshot := models.PolarCheckoutSnapshot{CheckoutID: "chk_1", URL: "https://buy.polar.sh/chk_1", Status: "open", ExpiresAt: &expires}
	require.NoError(t, intent.MarkOpen(db, snapshot))
	assert.Equal(t, models.BillingCheckoutStateOpen, intent.State)
	assert.Equal(t, "chk_1", intent.PolarCheckoutIDValue())
	assert.Equal(t, "https://buy.polar.sh/chk_1", intent.CheckoutURLValue())
	assert.True(t, intent.ExpiresAt.Equal(expires))
	assert.True(t, intent.Unresolved())

	require.ErrorIs(t, stale.MarkAmbiguous(db, "late timeout"), models.ErrBillingCheckoutStateConflict,
		"only a creating intent can become ambiguous")
	require.Error(t, intent.MarkOpen(db, models.PolarCheckoutSnapshot{}), "a Polar checkout id is required")

	require.NoError(t, intent.MarkOpen(db, models.PolarCheckoutSnapshot{CheckoutID: "chk_1", Status: "confirmed"}))
	assert.Equal(t, "https://buy.polar.sh/chk_1", intent.CheckoutURLValue(), "a partial snapshot keeps stored fields")
	assert.Equal(t, "confirmed", *intent.PolarStatus)

	require.Error(t, intent.Resolve(db, models.BillingCheckoutStateOpen, nil), "open is not a resolved state")
	require.NoError(t, intent.Resolve(db, models.BillingCheckoutStateExpired, &models.PolarCheckoutSnapshot{CheckoutID: "chk_1", Status: "expired"}))
	assert.False(t, intent.Unresolved())
	require.NotNil(t, intent.ResolvedAt)

	require.ErrorIs(t, stale.Release(db, "late"), models.ErrBillingCheckoutStateConflict)
	require.ErrorIs(t, intent.MarkOpen(db, snapshot), models.ErrBillingCheckoutStateConflict)
	require.ErrorIs(t, intent.Resolve(db, models.BillingCheckoutStateSucceeded, nil), models.ErrBillingCheckoutStateConflict)
	stored, err := models.FindUnresolvedOrganizationBillingCheckout(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Nil(t, stored)
}

func Test__AmbiguousBillingCheckoutKeepsBlockingUntilResolved(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	intent := claimBillingCheckout(t, r.Organization.ID, "prod_team", nil)
	require.NoError(t, intent.MarkAmbiguous(db, "502 from Polar"))

	again, claimed, err := models.ClaimOrganizationBillingCheckout(db, r.Organization.ID, "team", "prod_team", nil)
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Equal(t, intent.ID, again.ID)
	assert.Equal(t, models.BillingCheckoutStateAmbiguous, again.State)
	assert.Equal(t, "502 from Polar", *again.LastError)

	require.NoError(t, again.MarkOpen(db, models.PolarCheckoutSnapshot{CheckoutID: "chk_found", URL: "https://buy.polar.sh/chk_found", Status: "open"}))
	assert.Equal(t, models.BillingCheckoutStateOpen, again.State)
}

func Test__MatchesSelectionComparesPlanProductAndSeats(t *testing.T) {
	intent := &models.OrganizationBillingCheckout{PlanKey: "team", PolarProductID: "prod_team", Seats: new(3)}
	assert.True(t, intent.MatchesSelection("team", "prod_team", new(3)))
	assert.False(t, intent.MatchesSelection("team", "prod_team", new(4)))
	assert.False(t, intent.MatchesSelection("team", "prod_team", nil))
	assert.False(t, intent.MatchesSelection("business", "prod_team", new(3)))
	assert.False(t, intent.MatchesSelection("team", "prod_other", new(3)))
	flat := &models.OrganizationBillingCheckout{PlanKey: "business", PolarProductID: "prod_business"}
	assert.True(t, flat.MatchesSelection("business", "prod_business", nil))
	assert.False(t, flat.MatchesSelection("business", "prod_business", new(1)))
}

func Test__SubscriptionResolvesOnlyTheMatchingProductIntent(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	intent := claimBillingCheckout(t, r.Organization.ID, "prod_team", new(3))
	require.NoError(t, intent.MarkOpen(db, models.PolarCheckoutSnapshot{CheckoutID: "chk_1", Status: "open"}))

	require.NoError(t, models.ResolveOrganizationBillingCheckoutForSubscription(db, r.Organization.ID, "prod_other"))
	stored, err := models.FindUnresolvedOrganizationBillingCheckout(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)

	require.NoError(t, models.ResolveOrganizationBillingCheckoutForSubscription(db, r.Organization.ID, "prod_team"))
	stored, err = models.FindUnresolvedOrganizationBillingCheckout(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Nil(t, stored)
	var resolved models.OrganizationBillingCheckout
	require.NoError(t, db.First(&resolved, "id = ?", intent.ID).Error)
	assert.Equal(t, models.BillingCheckoutStateSucceeded, resolved.State)
	assert.NotNil(t, resolved.ResolvedAt)
}

func Test__PreexistingCheckoutIsAdoptedOnlyForAnOrganizationWithoutIntents(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	snapshot := models.PolarCheckoutSnapshot{CheckoutID: "chk_legacy", URL: "https://buy.polar.sh/chk_legacy", Status: "open"}

	adopted, err := models.AdoptPreexistingOrganizationBillingCheckout(db, r.Organization.ID, "team", "prod_team", new(3), snapshot)
	require.NoError(t, err)
	require.NotNil(t, adopted)
	assert.Equal(t, models.BillingCheckoutStateOpen, adopted.State)
	assert.Equal(t, "chk_legacy", adopted.PolarCheckoutIDValue())
	assert.True(t, adopted.MatchesSelection("team", "prod_team", new(3)))

	require.NoError(t, adopted.Resolve(db, models.BillingCheckoutStateExpired, nil))
	again, err := models.AdoptPreexistingOrganizationBillingCheckout(db, r.Organization.ID, "team", "prod_team", new(3),
		models.PolarCheckoutSnapshot{CheckoutID: "chk_legacy_2", Status: "open"})
	require.NoError(t, err)
	assert.Nil(t, again, "rollout adoption happens once, before the first intent")

	_, err = models.AdoptPreexistingOrganizationBillingCheckout(db, r.Organization.ID, "team", "prod_team", nil, models.PolarCheckoutSnapshot{})
	require.Error(t, err, "an adopted checkout needs its real Polar id")
}
