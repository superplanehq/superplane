package entitlements

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"testing"
	"time"
)

func TestResolveCatalogAccessAndMarkup(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	var original models.PlanFeatureDefinition
	require.NoError(t, db.First(&original, "key = ?", "byo_key").Error)
	t.Cleanup(func() { require.NoError(t, models.UpdatePlanFeatureDefinition(db, &original)) })
	require.NoError(t, models.UpdatePlanFeatureDefinition(db, &models.PlanFeatureDefinition{Key: "byo_key", Label: "Bring your own API key", Gated: true}))
	seats, parallel, markup := 3, 2, 250
	p := models.BillingPlan{Key: "test_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", SeatLimit: &seats, ParallelTaskLimit: &parallel, MarkupBPS: &markup, CreditFrequency: "once", CreditExpiration: "never", CreditAmountCents: 1000}
	require.NoError(t, models.SaveBillingPlan(db, &p))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, p.Key, nil)
	require.NoError(t, err)
	resolved, err := Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, 3, *resolved.SeatLimit)
	require.Equal(t, 2, *resolved.ParallelTaskLimit)
	require.Equal(t, 250, resolved.MarkupBPS)
	require.True(t, resolved.HostedExecutionAllowed)
	require.True(t, resolved.Features["byo_subscription"])
	require.False(t, resolved.Features["byo_key"])
	require.NoError(t, models.SetOrganizationFeatureGrant(db, models.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: true}))
	enabled, err := HasFeature(db, r.Organization.ID, "byo_key")
	require.NoError(t, err)
	require.True(t, enabled)
	p.Features = []string{"byo_key"}
	require.NoError(t, models.SaveBillingPlan(db, &p))
	require.NoError(t, models.SetOrganizationFeatureGrant(db, models.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: false}))
	enabled, err = HasFeature(db, r.Organization.ID, "byo_key")
	require.NoError(t, err)
	require.True(t, enabled)
	zero := 0
	require.NoError(t, models.UpsertOrganizationLLMMarkup(db, r.Organization.ID, &zero))
	resolved, err = Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Zero(t, resolved.MarkupBPS)
	p.Features = []string{}
	p.MarkupBPS = nil
	require.NoError(t, models.SaveBillingPlan(db, &p))
	require.NoError(t, models.UpsertOrganizationLLMMarkup(db, r.Organization.ID, nil))
	installation, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	resolved, err = Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, installation.MarkupBPS, resolved.MarkupBPS)
	t.Setenv(models.BillingEnabledEnv, "no")
	resolved, err = Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Nil(t, resolved.SeatLimit)
	require.Nil(t, resolved.ParallelTaskLimit)
	definitions, err := models.ListPlanFeatureDefinitions(db)
	require.NoError(t, err)
	for _, feature := range definitions {
		require.True(t, resolved.Features[feature.Key])
	}
}

func TestSeatOverridePrecedenceAndPersistence(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.Conn()
	t.Setenv(models.BillingEnabledEnv, "yes")
	planLimit := 4
	plan := models.BillingPlan{Key: "seat_" + uuid.NewString()[:8], Name: "Seats", Category: "paid", Visibility: "private", SeatLimit: &planLimit, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	check := func(expected *int) {
		t.Helper()
		effective, err := Resolve(db, r.Organization.ID)
		require.NoError(t, err)
		require.Equal(t, expected, effective.SeatLimit)
	}
	check(&planLimit)
	custom := 2
	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, &custom))
	check(&custom)
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, "none", nil)
	require.NoError(t, err)
	check(&custom)
	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, nil))
	check(nil)
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	check(nil)
	require.NoError(t, models.DeleteOrganizationSeatLimitOverride(db, r.Organization.ID))
	check(&planLimit)
	t.Setenv(models.BillingEnabledEnv, "no")
	check(nil)
}

func TestResolvePurchasedSeats(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "yes")
	db := database.Conn()
	maximum := 50
	plan := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: &maximum, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 3).Error)
	member := support.CreateUser(t, r, r.Organization.ID)
	removed := support.CreateUser(t, r, r.Organization.ID)
	require.NoError(t, db.Delete(removed).Error)
	_, err = models.CreateAPIKey(db, r.Organization.ID, "service", nil, r.User, nil, nil)
	require.NoError(t, err)

	expected := SeatUsage{Invitable: seatCount(1), PerSeat: true, Purchased: seatCount(3), Occupied: 2, Available: seatCount(1), PlanMaximum: seatCount(50), Purchasable: seatCount(47), RenewalQuantity: seatCount(2)}
	check := func() {
		t.Helper()
		actual, err := ResolveSeats(db, r.Organization.ID)
		require.NoError(t, err)
		require.Equal(t, expected, *actual)
		require.Equal(t, expected.Purchased, actual.JoinLimit())
	}
	check()
	maximum = 60
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	expected.PlanMaximum, expected.Purchasable = seatCount(60), seatCount(57)
	check()
	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, seatCount(80)))
	expected.PlanMaximum, expected.Purchasable = seatCount(80), seatCount(77)
	check()
	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, nil))
	expected.PlanMaximum, expected.Purchasable = nil, nil
	check()
	require.NoError(t, models.DeleteOrganizationSeatLimitOverride(db, r.Organization.ID))
	expected.PlanMaximum, expected.Purchasable = seatCount(60), seatCount(57)
	check()
	// Simulate a stored plan that predates the required per-seat maximum.
	require.NoError(t, db.Model(&plan).Update("seat_limit", nil).Error)
	expected.PlanMaximum, expected.Purchasable = nil, nil
	check()
	plan.SeatLimit = seatCount(1)
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	expected.PlanMaximum, expected.Purchasable = seatCount(1), seatCount(0)
	check()

	for _, quantity := range []*int{seatCount(1), seatCount(0), nil} {
		require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", quantity).Error)
		expected.Invitable = seatCount(0)
		expected.Purchased, expected.Available = quantity, seatCount(0)
		if quantity == nil {
			expected.Purchased = seatCount(0)
		}
		expected.Purchasable = seatCount(1 - *expected.Purchased)
		check()
	}
	require.NoError(t, db.Delete(member).Error)
	require.NoError(t, db.Delete(&models.User{}, "id = ?", r.User).Error)
	expected.Occupied, expected.RenewalQuantity = 0, seatCount(1)
	check()
	t.Setenv(models.BillingEnabledEnv, "no")
	actual, err := ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, SeatUsage{}, *actual)
	require.Nil(t, actual.JoinLimit())
}

func TestResolveMemberLimits(t *testing.T) {
	for _, key := range []string{"business_legacy", "business_manual", "trial", "none", "finite"} {
		t.Run(key, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "yes")
			db := database.Conn()
			var catalog *models.BillingPlan
			if key == "finite" {
				catalog = &models.BillingPlan{Key: "finite_" + uuid.NewString()[:8], Name: "Finite", Category: "paid", Visibility: "private", SeatLimit: seatCount(7), CreditFrequency: "none", CreditExpiration: "never"}
				require.NoError(t, models.SaveBillingPlan(db, catalog))
			} else {
				var err error
				catalog, err = models.FindBillingPlan(db, key)
				require.NoError(t, err)
			}
			// Resolve the plan and member count from the caller's uncommitted changes.
			tx := db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			end := time.Now().Add(time.Hour)
			require.NoError(t, tx.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{"plan_id": catalog.ID, "plan": catalog.ResponsePlan(), "plan_source": "admin", "trial_ends_at": end, "purchased_seats": 99}).Error)
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
					require.NoError(t, models.SetOrganizationSeatLimitOverride(tx, r.Organization.ID, tc.limit))
				}
				if tc.name == "deleted override" {
					require.NoError(t, models.DeleteOrganizationSeatLimitOverride(tx, r.Organization.ID))
				}
				actual, err := ResolveSeats(tx, r.Organization.ID)
				require.NoError(t, err)
				require.Equal(t, SeatUsage{Occupied: 1, MemberLimit: tc.limit}, *actual, tc.name)
				require.Equal(t, tc.limit, actual.JoinLimit())
			}
			require.NoError(t, tx.Delete(&models.User{}, "id = ?", r.User).Error)
			actual, err := ResolveSeats(tx, r.Organization.ID)
			require.NoError(t, err)
			require.Zero(t, actual.Occupied)
			t.Setenv(models.BillingEnabledEnv, "no")
			actual, err = ResolveSeats(tx, r.Organization.ID)
			require.NoError(t, err)
			require.Equal(t, SeatUsage{}, *actual)
			require.Nil(t, actual.JoinLimit())
		})
	}
}

func TestResolveSeatsForPlanUsesLoadedPlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "yes")
	db := database.Conn()
	plan := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: seatCount(50), CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 3).Error)
	loaded, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	now := time.Now()
	invitation := models.OrganizationInvitation{OrganizationID: r.Organization.ID, Email: "pending@example.com", Role: models.RoleOrgOperator, InvitedBy: r.User, State: models.InvitationStatePending, SentAt: now, ExpiresAt: now.Add(time.Hour)}
	require.NoError(t, invitation.Save(db))

	// Change the stored plan after the caller's read. Seat usage must follow
	// the loaded plan so counts and prices stay consistent.
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 9).Error)

	actual, err := ResolveSeatsForPlan(db, r.Organization.ID, loaded)
	require.NoError(t, err)
	require.Equal(t, SeatUsage{PendingInvitations: 1, Invitable: seatCount(1), PerSeat: true, Purchased: seatCount(3), Occupied: 1, Available: seatCount(2), PlanMaximum: seatCount(50), Purchasable: seatCount(47), RenewalQuantity: seatCount(1)}, *actual)

	current, err := ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, 1, current.PendingInvitations)
	require.Equal(t, seatCount(9), current.Purchased)
	require.Equal(t, seatCount(7), current.Invitable)
}

func seatCount(quantity int) *int { return &quantity }
