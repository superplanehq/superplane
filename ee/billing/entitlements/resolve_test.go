package entitlements_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestResolveCatalogAccessAndMarkup(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	var original billingmodels.PlanFeatureDefinition
	require.NoError(t, db.First(&original, "key = ?", "byo_key").Error)
	t.Cleanup(func() { require.NoError(t, billingmodels.UpdatePlanFeatureDefinition(db, &original)) })
	require.NoError(t, billingmodels.UpdatePlanFeatureDefinition(db, &billingmodels.PlanFeatureDefinition{Key: "byo_key", Label: "Bring your own API key", Gated: true}))
	seats, parallel, markup := 3, 2, 250
	p := billingmodels.BillingPlan{Key: "test_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", SeatLimit: &seats, ParallelTaskLimit: &parallel, MarkupBPS: &markup, CreditFrequency: "once", CreditExpiration: "never", CreditAmountCents: 1000}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &p))
	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, p.Key, nil)
	require.NoError(t, err)
	resolved, err := entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, 3, *resolved.SeatLimit)
	require.Equal(t, 2, *resolved.ParallelTaskLimit)
	require.Equal(t, 250, resolved.MarkupBPS)
	require.True(t, resolved.HostedExecutionAllowed)
	require.True(t, resolved.Features["byo_subscription"])
	require.False(t, resolved.Features["byo_key"])
	require.NoError(t, billingmodels.SetOrganizationFeatureGrant(db, billingmodels.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: true}))
	enabled, err := entitlements.HasFeature(db, r.Organization.ID, "byo_key")
	require.NoError(t, err)
	require.True(t, enabled)
	p.Features = []string{"byo_key"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &p))
	require.NoError(t, billingmodels.SetOrganizationFeatureGrant(db, billingmodels.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: false}))
	enabled, err = entitlements.HasFeature(db, r.Organization.ID, "byo_key")
	require.NoError(t, err)
	require.True(t, enabled)
	zero := 0
	require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(db, r.Organization.ID, &zero))
	resolved, err = entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Zero(t, resolved.MarkupBPS)
	p.Features = []string{}
	p.MarkupBPS = nil
	require.NoError(t, billingmodels.SaveBillingPlan(db, &p))
	require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(db, r.Organization.ID, nil))
	installation, err := billingmodels.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	resolved, err = entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, installation.MarkupBPS, resolved.MarkupBPS)
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	resolved, err = entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Nil(t, resolved.SeatLimit)
	require.Nil(t, resolved.ParallelTaskLimit)
	definitions, err := billingmodels.ListPlanFeatureDefinitions(db)
	require.NoError(t, err)
	for _, feature := range definitions {
		require.True(t, resolved.Features[feature.Key])
	}
}

func TestSeatOverridePrecedenceAndPersistence(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.Conn()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	planLimit := 4
	plan := billingmodels.BillingPlan{Key: "seat_" + uuid.NewString()[:8], Name: "Seats", Category: "paid", Visibility: "private", SeatLimit: &planLimit, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	check := func(expected *int) {
		t.Helper()
		effective, err := entitlements.Resolve(db, r.Organization.ID)
		require.NoError(t, err)
		require.Equal(t, expected, effective.SeatLimit)
	}
	check(&planLimit)
	custom := 2
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, &custom))
	check(&custom)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, "none", nil)
	require.NoError(t, err)
	check(&custom)
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, nil))
	check(nil)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	check(nil)
	require.NoError(t, billingmodels.DeleteOrganizationSeatLimitOverride(db, r.Organization.ID))
	check(&planLimit)
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	check(nil)
}

func TestResolvePurchasedSeats(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	maximum := 50
	plan := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: &maximum, PolarProductID: new("prod_" + uuid.NewString()), CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	newSubscriptionReplay(t, r.Organization.ID, &plan).apply("active", seatCount(3))
	member := support.CreateUser(t, r, r.Organization.ID)
	removed := support.CreateUser(t, r, r.Organization.ID)
	require.NoError(t, db.Delete(removed).Error)
	_, err := models.CreateAPIKey(db, r.Organization.ID, "service", nil, r.User, nil, nil)
	require.NoError(t, err)

	expected := entitlements.SeatUsage{Invitable: seatCount(1), PerSeat: true, Purchased: seatCount(3), Occupied: 2, Available: seatCount(1), PlanMaximum: seatCount(50), Purchasable: seatCount(47), RenewalQuantity: seatCount(2)}
	check := func() {
		t.Helper()
		actual, err := entitlements.ResolveSeats(db, r.Organization.ID)
		require.NoError(t, err)
		require.Equal(t, expected, *actual)
		require.Equal(t, expected.Purchased, entitlements.JoinLimit(actual))
	}
	check()
	maximum = 60
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	expected.PlanMaximum, expected.Purchasable = seatCount(60), seatCount(57)
	check()
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, seatCount(80)))
	expected.PlanMaximum, expected.Purchasable = seatCount(80), seatCount(77)
	check()
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, nil))
	expected.PlanMaximum, expected.Purchasable = nil, nil
	check()
	require.NoError(t, billingmodels.DeleteOrganizationSeatLimitOverride(db, r.Organization.ID))
	expected.PlanMaximum, expected.Purchasable = seatCount(60), seatCount(57)
	check()
	// Simulate a stored plan that predates the required per-seat maximum.
	require.NoError(t, db.Model(&plan).Update("seat_limit", nil).Error)
	expected.PlanMaximum, expected.Purchasable = nil, nil
	check()
	plan.SeatLimit = seatCount(1)
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	expected.PlanMaximum, expected.Purchasable = seatCount(1), seatCount(0)
	check()

	for _, quantity := range []*int{seatCount(1), seatCount(0), nil} {
		require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", quantity).Error)
		expected.Invitable = seatCount(0)
		expected.Purchased, expected.Available = quantity, seatCount(0)
		// The renewal never adds seats, and no paid seats give no renewal count.
		expected.RenewalQuantity = quantity
		if quantity == nil || *quantity == 0 {
			expected.Purchased, expected.RenewalQuantity = seatCount(0), nil
		}
		expected.Purchasable = seatCount(1 - *expected.Purchased)
		check()
	}
	require.NoError(t, db.Delete(member).Error)
	require.NoError(t, db.Delete(&models.User{}, "id = ?", r.User).Error)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 3).Error)
	expected.Occupied, expected.RenewalQuantity = 0, seatCount(1)
	expected.Purchased, expected.Available, expected.Invitable, expected.Purchasable = seatCount(3), seatCount(3), seatCount(3), seatCount(0)
	check()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	actual, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, entitlements.SeatUsage{}, *actual)
	require.Nil(t, entitlements.JoinLimit(actual))
}

func TestResolveMemberLimits(t *testing.T) {
	for _, key := range []string{"business_legacy", "business_manual", "trial", "none", "finite"} {
		t.Run(key, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			db := database.Conn()
			var catalog *billingmodels.BillingPlan
			if key == "finite" {
				catalog = &billingmodels.BillingPlan{Key: "finite_" + uuid.NewString()[:8], Name: "Finite", Category: "paid", Visibility: "private", SeatLimit: seatCount(7), CreditFrequency: "none", CreditExpiration: "never"}
				require.NoError(t, billingmodels.SaveBillingPlan(db, catalog))
			} else {
				var err error
				catalog, err = billingmodels.FindBillingPlan(db, key)
				require.NoError(t, err)
			}
			// Resolve the plan and member count from the caller's uncommitted changes.
			tx := db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			end := time.Now().Add(time.Hour)
			require.NoError(t, tx.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{"plan_id": catalog.ID, "plan": catalog.ResponsePlan(), "plan_source": "admin", "trial_ends_at": end, "purchased_seats": 99}).Error)
			for _, tc := range []struct {
				name  string
				limit *int
			}{
				{"catalog", catalog.SeatLimit},
				{"override", seatCount(2)},
				{"unlimited override", nil},
				{"deleted override", catalog.SeatLimit},
			} {
				if tc.name == "override" || tc.name == "unlimited override" {
					require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(tx, r.Organization.ID, tc.limit))
				}
				if tc.name == "deleted override" {
					require.NoError(t, billingmodels.DeleteOrganizationSeatLimitOverride(tx, r.Organization.ID))
				}
				actual, err := entitlements.ResolveSeats(tx, r.Organization.ID)
				require.NoError(t, err)
				require.Equal(t, entitlements.SeatUsage{Occupied: 1, MemberLimit: tc.limit}, *actual, tc.name)
				require.Equal(t, tc.limit, entitlements.JoinLimit(actual))
			}
			require.NoError(t, tx.Delete(&models.User{}, "id = ?", r.User).Error)
			actual, err := entitlements.ResolveSeats(tx, r.Organization.ID)
			require.NoError(t, err)
			require.Zero(t, actual.Occupied)
			t.Setenv(billingmodels.BillingEnabledEnv, "no")
			actual, err = entitlements.ResolveSeats(tx, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, entitlements.SeatUsage{}, *actual)
			require.Nil(t, entitlements.JoinLimit(actual))
		})
	}
}

func TestResolveSeatsForPlanUsesLoadedPlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	plan := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: seatCount(50), PolarProductID: new("prod_" + uuid.NewString()), CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	newSubscriptionReplay(t, r.Organization.ID, &plan).apply("active", seatCount(3))
	loaded, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	now := time.Now()
	invitation := models.OrganizationInvitation{OrganizationID: r.Organization.ID, Email: "pending@example.com", Role: models.RoleOrgOperator, InvitedBy: r.User, State: models.InvitationStatePending, SentAt: now, ExpiresAt: now.Add(time.Hour)}
	require.NoError(t, invitation.Save(db))

	// Change the stored plan after the caller's read. Seat usage must follow
	// the loaded plan so counts and prices stay consistent.
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 9).Error)

	actual, err := entitlements.ResolveSeatsForPlan(db, r.Organization.ID, loaded)
	require.NoError(t, err)
	require.Equal(t, entitlements.SeatUsage{PendingInvitations: 1, Invitable: seatCount(1), PerSeat: true, Purchased: seatCount(3), Occupied: 1, Available: seatCount(2), PlanMaximum: seatCount(50), Purchasable: seatCount(47), RenewalQuantity: seatCount(1)}, *actual)

	current, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, 1, current.PendingInvitations)
	require.Equal(t, seatCount(9), current.Purchased)
	require.Equal(t, seatCount(7), current.Invitable)
}

// subscriptionReplay delivers Polar subscription webhooks in order, as they arrive.
type subscriptionReplay struct {
	t              *testing.T
	orgID          uuid.UUID
	productID      string
	subscriptionID string
	modifiedAt     time.Time
	// pastDueAt is sent as past_due_at when set.
	pastDueAt *time.Time
}

func newSubscriptionReplay(t *testing.T, orgID uuid.UUID, catalog *billingmodels.BillingPlan) *subscriptionReplay {
	return &subscriptionReplay{t: t, orgID: orgID, productID: *catalog.PolarProductID, subscriptionID: "sub_" + uuid.NewString(), modifiedAt: time.Now().Add(-time.Hour)}
}

func (s *subscriptionReplay) apply(status string, seats *int) {
	s.t.Helper()
	s.modifiedAt = s.modifiedAt.Add(time.Minute)
	start := time.Now().Add(-7 * 24 * time.Hour)
	data := map[string]any{
		"id": s.subscriptionID, "product_id": s.productID, "status": status, "seats": seats,
		"external_customer_id": s.orgID.String(), "modified_at": s.modifiedAt.UTC().Format(time.RFC3339),
		"current_period_start": start.UTC().Format(time.RFC3339), "current_period_end": start.AddDate(0, 1, 0).UTC().Format(time.RFC3339),
	}
	if s.pastDueAt != nil {
		data["past_due_at"] = s.pastDueAt.UTC().Format(time.RFC3339)
	}
	body, err := json.Marshal(map[string]any{"type": "subscription.updated", "data": data})
	require.NoError(s.t, err)
	var event polar.SubscriptionWebhookEvent
	require.NoError(s.t, json.Unmarshal(body, &event))
	require.NoError(s.t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
}

func savePaidCatalog(t *testing.T, perSeat bool) *billingmodels.BillingPlan {
	t.Helper()
	product := "prod_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{Key: "paid_" + uuid.NewString()[:8], Name: "Paid", Category: "paid", Visibility: "private", PerSeat: perSeat, PriceCents: 2000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	if perSeat {
		catalog.SeatLimit = seatCount(50)
	}
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	return &catalog
}

func TestPastDueLocksSeatsOfSeatBasedSubscription(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	support.CreateUser(t, r, r.Organization.ID)
	replay := newSubscriptionReplay(t, r.Organization.ID, savePaidCatalog(t, true))
	resolve := func() *entitlements.SeatUsage {
		t.Helper()
		usage, err := entitlements.ResolveSeats(db, r.Organization.ID)
		require.NoError(t, err)
		return usage
	}

	replay.apply("active", seatCount(2))
	active := resolve()
	require.False(t, active.PaymentBlocked)
	require.Equal(t, 0, *active.Invitable)
	require.Equal(t, 48, *active.Purchasable)

	replay.apply("past_due", seatCount(2))
	blocked := resolve()
	require.True(t, blocked.PaymentBlocked)
	require.Equal(t, 2, blocked.Occupied)
	require.Equal(t, seatCount(2), blocked.Purchased)
	require.Equal(t, seatCount(2), entitlements.JoinLimit(blocked))
	require.Equal(t, seatCount(0), blocked.Invitable)
	require.Equal(t, seatCount(0), blocked.Purchasable)

	replay.apply("active", seatCount(2))
	require.Equal(t, *active, *resolve(), "an active snapshot restores the normal seat rules")
}

func TestPausedSubscriptionLocksSeats(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	support.CreateUser(t, r, r.Organization.ID)
	replay := newSubscriptionReplay(t, r.Organization.ID, savePaidCatalog(t, true))

	replay.apply("active", seatCount(5))
	replay.apply("paused", seatCount(5))

	usage, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.True(t, usage.PaymentBlocked)
	require.Equal(t, seatCount(5), usage.Purchased)
	require.Equal(t, seatCount(2), entitlements.JoinLimit(usage))
	require.Equal(t, seatCount(0), usage.Invitable)
}

func TestPastDueGraceKeepsTeamPlanButLocksSeats(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	support.CreateUser(t, r, r.Organization.ID)
	require.NoError(t, billingmodels.UpdatePastDueGraceDays(db, 7))
	replay := newSubscriptionReplay(t, r.Organization.ID, savePaidCatalog(t, true))
	replay.apply("active", seatCount(5))

	failure := time.Now().Add(-24 * time.Hour)
	replay.pastDueAt = &failure
	replay.apply("past_due", seatCount(5))

	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.True(t, plan.Catalog.PerSeat, "Team features stay on during grace")
	usage, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.True(t, usage.PaymentBlocked)
	require.Equal(t, seatCount(2), entitlements.JoinLimit(usage))
	require.Equal(t, seatCount(0), usage.Invitable)

	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).
		Update("past_due_since", time.Now().Add(-8*24*time.Hour)).Error)
	lapsed, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.True(t, lapsed.PaymentBlocked, "the seat lock outlives the grace period")
	require.Equal(t, seatCount(2), entitlements.JoinLimit(lapsed))
}

func TestPastDueLockIgnoresSubscriptionWithoutSeats(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	replay := newSubscriptionReplay(t, r.Organization.ID, savePaidCatalog(t, false))

	replay.apply("active", nil)
	replay.apply("past_due", nil)
	usage, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.False(t, usage.PaymentBlocked)
	require.Equal(t, usage.MemberLimit, entitlements.JoinLimit(usage))
}

func TestUnpaidAndRevokedSnapshotsEndThePastDueLock(t *testing.T) {
	for _, status := range []string{"unpaid", "revoked"} {
		t.Run(status, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			db := database.Conn()
			support.CreateUser(t, r, r.Organization.ID)
			replay := newSubscriptionReplay(t, r.Organization.ID, savePaidCatalog(t, true))
			replay.apply("active", seatCount(2))
			replay.apply("past_due", seatCount(2))
			blocked, err := entitlements.ResolveSeats(db, r.Organization.ID)
			require.NoError(t, err)
			require.True(t, blocked.PaymentBlocked)
			blockedEntitlements, err := entitlements.Resolve(db, r.Organization.ID)
			require.NoError(t, err)

			replay.apply(status, seatCount(2))
			lapsed, err := entitlements.ResolveSeats(db, r.Organization.ID)
			require.NoError(t, err)
			require.False(t, lapsed.PaymentBlocked)
			require.False(t, lapsed.PerSeat)
			require.Equal(t, lapsed.MemberLimit, entitlements.JoinLimit(lapsed))
			require.Nil(t, lapsed.Invitable)

			// Past due keeps today's plan change, so the lapse changes no entitlement.
			lapsedEntitlements, err := entitlements.Resolve(db, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, lapsedEntitlements, blockedEntitlements)
		})
	}
}

func seatCount(quantity int) *int { return &quantity }
