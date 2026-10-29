package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

// useActivationCatalogPlan makes the catalog plan with an activation key match
// the test. Recording reads the exact key, so the test cannot use another key.
// The test database can already hold the key from the seed migration, so the
// helper saves the row, applies the test values, and restores the row at
// cleanup. A missing row is created and removed at cleanup.
func useActivationCatalogPlan(t *testing.T, key string, plan billingmodels.BillingPlan) {
	t.Helper()
	db := database.Conn()
	cleanupDB := database.DB(context.WithoutCancel(t.Context()))
	existing, err := billingmodels.FindBillingPlan(db, key)
	if err != nil {
		plan.Key, plan.Name, plan.Visibility = key, key, "private"
		plan.CreditFrequency, plan.CreditExpiration = "none", "never"
		require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
		t.Cleanup(func() {
			require.NoError(t, cleanupDB.Where("plan_id = ?", plan.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
			require.NoError(t, cleanupDB.Delete(&billingmodels.BillingPlan{}, "id = ?", plan.ID).Error)
		})
		return
	}
	original := map[string]any{
		"category":               existing.Category,
		"per_seat":               existing.PerSeat,
		"price_cents":            existing.PriceCents,
		"seat_limit":             existing.SeatLimit,
		"require_payment_method": existing.RequirePaymentMethod,
		"polar_product_id":       existing.PolarProductID,
	}
	require.NoError(t, db.Model(existing).Updates(map[string]any{
		"category":               plan.Category,
		"per_seat":               plan.PerSeat,
		"price_cents":            plan.PriceCents,
		"seat_limit":             plan.SeatLimit,
		"require_payment_method": plan.RequirePaymentMethod,
		"polar_product_id":       plan.PolarProductID,
	}).Error)
	t.Cleanup(func() {
		require.NoError(t, cleanupDB.Model(existing).Updates(original).Error)
	})
}

func activationKeys(t *testing.T, orgID uuid.UUID) []string {
	t.Helper()
	var rows []billingmodels.PlanActivation
	require.NoError(t, database.Conn().Where("organization_id = ?", orgID).Order("activated_at, id").Find(&rows).Error)
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.PlanKey)
	}
	return keys
}

func applyPolarApplied(t *testing.T, db *gorm.DB, orgID uuid.UUID, sub billingmodels.PolarSubscriptionApply) {
	t.Helper()
	result, err := billingmodels.ApplyPolarSubscription(db, orgID, sub, nil)
	require.NoError(t, err)
	require.Equal(t, billingmodels.PolarSubscriptionApplied, result.Disposition)
}

func Test__PlanActivationRecordsInitialSwitchAndReactivationOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	teamProduct := "prod_team_" + uuid.NewString()
	starterProduct := "prod_starter_" + uuid.NewString()
	useActivationCatalogPlan(t, billingmodels.PlanActivationKeyTeam, billingmodels.BillingPlan{
		Category: "paid", PerSeat: true, PriceCents: 5000, SeatLimit: new(50), PolarProductID: &teamProduct,
	})
	useActivationCatalogPlan(t, billingmodels.PlanActivationKeyStarter, billingmodels.BillingPlan{
		Category: "paid", PriceCents: 2000, SeatLimit: new(10), PolarProductID: &starterProduct,
	})

	start := time.Now().Add(-7 * day)
	end := time.Now().Add(23 * day)
	modified := time.Now().Add(-time.Hour)
	seats := new(3)
	team := billingmodels.PolarSubscriptionApply{
		ID: "sub_activation", ProductID: teamProduct, Status: billingmodels.PolarSubscriptionStatusActive,
		PeriodStart: &start, PeriodEnd: &end, ModifiedAt: &modified, PurchasedSeats: seats,
	}

	applyPolarApplied(t, db, r.Organization.ID, team)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyTeam}, activationKeys(t, r.Organization.ID))

	applyPolarApplied(t, db, r.Organization.ID, team)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyTeam}, activationKeys(t, r.Organization.ID), "replayed snapshot")

	renewedEnd := end.AddDate(0, 1, 0)
	renewedModified := modified.Add(time.Minute)
	renewed := team
	renewed.PeriodStart, renewed.PeriodEnd, renewed.ModifiedAt = &end, &renewedEnd, &renewedModified
	applyPolarApplied(t, db, r.Organization.ID, renewed)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyTeam}, activationKeys(t, r.Organization.ID), "renewal")

	switchedModified := renewedModified.Add(time.Minute)
	starter := renewed
	starter.ProductID, starter.PurchasedSeats, starter.ModifiedAt = starterProduct, nil, &switchedModified
	applyPolarApplied(t, db, r.Organization.ID, starter)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyTeam, billingmodels.PlanActivationKeyStarter},
		activationKeys(t, r.Organization.ID), "switch from team to starter")

	canceledModified := switchedModified.Add(time.Minute)
	canceled := starter
	canceled.Status, canceled.ModifiedAt = billingmodels.PolarSubscriptionStatusCanceled, &canceledModified
	applyPolarApplied(t, db, r.Organization.ID, canceled)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyTeam, billingmodels.PlanActivationKeyStarter},
		activationKeys(t, r.Organization.ID), "cancellation is not an activation")

	reactivatedModified := canceledModified.Add(time.Minute)
	reactivated := team
	reactivated.ID, reactivated.Verified = "sub_reactivated", true
	reactivated.ModifiedAt, reactivated.PeriodEnd = &reactivatedModified, &renewedEnd
	applyPolarApplied(t, db, r.Organization.ID, reactivated)
	assert.Equal(t, []string{
		billingmodels.PlanActivationKeyTeam,
		billingmodels.PlanActivationKeyStarter,
		billingmodels.PlanActivationKeyTeam,
	}, activationKeys(t, r.Organization.ID), "reactivation counts again")

	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.PlanActivationKeyTeam, plan.Catalog.Key)
}

func Test__PlanActivationRecordsRejectedAndRecoveredSnapshotsCorrectly(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	teamProduct := "prod_team_" + uuid.NewString()
	useActivationCatalogPlan(t, billingmodels.PlanActivationKeyTeam, billingmodels.BillingPlan{
		Category: "paid", PerSeat: true, PriceCents: 5000, SeatLimit: new(50), PolarProductID: &teamProduct,
	})
	setPastDueGraceDays(t, 7)

	start := time.Now().Add(-7 * day)
	end := time.Now().Add(23 * day)
	modified := time.Now().Add(-time.Hour)
	// Seat-based products need a seat count. This snapshot is rejected before
	// any write, so it records nothing.
	rejected := billingmodels.PolarSubscriptionApply{
		ID: "sub_recovery", ProductID: teamProduct, Status: billingmodels.PolarSubscriptionStatusActive,
		PeriodStart: &start, PeriodEnd: &end, ModifiedAt: &modified,
	}
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, rejected, nil)
	require.ErrorIs(t, err, billingmodels.ErrInvalidPurchasedSeats)
	assert.Empty(t, activationKeys(t, r.Organization.ID))

	// A plan that already has access when the feature ships is not an initial activation.
	existingSubscription := "sub_existing"
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
		"plan": billingmodels.PlanActivationKeyTeam, "plan_id": mustPlanID(t, billingmodels.PlanActivationKeyTeam),
		"plan_source": billingmodels.BillingPlanSourcePolar, "polar_subscription_id": existingSubscription,
		"polar_subscription_status": billingmodels.PolarSubscriptionStatusActive, "current_period_end": end,
		"purchased_seats": 3, "polar_modified_at": modified,
	}).Error)

	accepted := rejected
	accepted.ID, accepted.PurchasedSeats = existingSubscription, new(3)
	accepted.ModifiedAt = new(modified.Add(time.Minute))
	applyPolarApplied(t, db, r.Organization.ID, accepted)
	assert.Empty(t, activationKeys(t, r.Organization.ID), "existing active plan at rollout")

	// A failed renewal keeps access during the grace period. Recovery in that
	// period keeps the same access, so it is not a new activation.
	pastDue := accepted
	pastDue.Status = billingmodels.PolarSubscriptionStatusPastDue
	pastDueAt := time.Now()
	pastDue.PastDueAt, pastDue.ModifiedAt = &pastDueAt, new(accepted.ModifiedAt.Add(time.Minute))
	applyPolarApplied(t, db, r.Organization.ID, pastDue)
	recovered := accepted
	recovered.ModifiedAt = new(pastDue.ModifiedAt.Add(time.Minute))
	applyPolarApplied(t, db, r.Organization.ID, recovered)
	assert.Empty(t, activationKeys(t, r.Organization.ID), "recovery within grace")
}

func Test__PlanActivationRecordsAdminAssignmentsOncePerTransition(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	useActivationCatalogPlan(t, billingmodels.PlanActivationKeyPayAsYouGo, billingmodels.BillingPlan{
		Category: "paid", PriceCents: 0,
	})

	_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.PlanActivationKeyPayAsYouGo, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyPayAsYouGo}, activationKeys(t, r.Organization.ID))

	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.PlanActivationKeyPayAsYouGo, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{billingmodels.PlanActivationKeyPayAsYouGo}, activationKeys(t, r.Organization.ID), "repeated assignment")

	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.PlanActivationKeyPayAsYouGo, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		billingmodels.PlanActivationKeyPayAsYouGo,
		billingmodels.PlanActivationKeyPayAsYouGo,
	}, activationKeys(t, r.Organization.ID), "reactivation after none")

	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)
	var rows []billingmodels.PlanActivation
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Order("activated_at, id").Find(&rows).Error)
	require.Len(t, rows, 2)
	assert.True(t, rows[0].ActivatedAt.Before(rows[1].ActivatedAt))
}

func Test__PlanActivationIgnoresCatalogKeysOutsideTheTargetSet(t *testing.T) {
	r := support.Setup(t)
	setLegacyBillingPlanTestProduct(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	applyPolarApplied(t, db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
		ProductID: "prod_business", ID: "sub_business", Status: billingmodels.PolarSubscriptionStatusActive,
		PeriodStart: &now, PeriodEnd: &end,
	})
	assert.Empty(t, activationKeys(t, r.Organization.ID))
}

func mustPlanID(t *testing.T, key string) uuid.UUID {
	t.Helper()
	plan, err := billingmodels.FindBillingPlan(database.Conn(), key)
	require.NoError(t, err)
	return plan.ID
}

func setPastDueGraceDays(t *testing.T, days int) {
	t.Helper()
	db := database.Conn()
	original, err := billingmodels.GetPastDueGraceDays(db)
	require.NoError(t, err)
	require.NoError(t, billingmodels.UpdatePastDueGraceDays(db, days))
	t.Cleanup(func() {
		require.NoError(t, billingmodels.UpdatePastDueGraceDays(database.DB(context.WithoutCancel(t.Context())), original))
	})
}
