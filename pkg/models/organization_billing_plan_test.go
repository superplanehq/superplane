package models_test

import (
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

func Test__EnsureOrganizationBillingPlanIsTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.IsOpenTrial(time.Now()))
}

func Test__SetAdminOrganizationPlanBusinessSkipsPolarCancel(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)
	assert.True(t, plan.IsActiveBusiness())
	assert.True(t, plan.AllowsCreditPurchase())

	now := time.Now()
	end := now.AddDate(0, 1, 0)
	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		"sub_admin",
		models.PolarSubscriptionStatusCanceled,
		&now,
		&end,
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanBusiness, after.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, after.PlanSource)
}

func Test__SetAdminOrganizationPlanBusinessGrantsIncludedUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)
	assert.True(t, plan.CurrentPeriodEnd.After(*plan.CurrentPeriodStart))

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)

	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)
	summary, err = models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__SetAdminOrganizationPlanBusinessGrantsIncludedWhenPeriodWasMissing(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":                 models.BillingPlanBusiness,
			"plan_source":          models.BillingPlanSourceAdmin,
			"current_period_start": nil,
			"current_period_end":   nil,
		}).Error)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultIncludedGrantCents), summary.IncludedRemainingMicros)
}

func Test__SetAdminOrganizationPlanNoneAfterBusinessResubscribe(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanNone)
	require.NoError(t, err)
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanNone)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.AllowsCreditPurchase())

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
}

func Test__SetAdminOrganizationPlanTrialExpiresIncludedUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)
}

func Test__ApplyPolarSubscriptionCancelRestoresOpenTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		"sub_restore",
		models.PolarSubscriptionStatusActive,
		&now,
		&end,
	)
	require.NoError(t, err)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		"sub_restore",
		models.PolarSubscriptionStatusCanceled,
		&now,
		&end,
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanTrial, after.Plan)
	assert.True(t, after.IsOpenTrial(time.Now()))
	assert.False(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionCancelEndsPlanAfterTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		"sub_lapsed",
		models.PolarSubscriptionStatusActive,
		&now,
		&end,
	)
	require.NoError(t, err)

	ended := now.Add(-time.Hour)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Update("trial_ends_at", ended).Error)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		"sub_lapsed",
		models.PolarSubscriptionStatusCanceled,
		&now,
		&end,
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanNone, after.Plan)
	assert.False(t, after.IsOpenTrial(time.Now()))
	assert.False(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionIncompleteKeepsTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		"sub_incomplete",
		models.PolarSubscriptionStatusIncomplete,
		&now,
		&end,
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanTrial, after.Plan)
	assert.True(t, after.IsOpenTrial(time.Now()))
	assert.Equal(t, models.PolarSubscriptionStatusIncomplete, after.PolarSubscriptionStatus)
}

func Test__AssertHostedRunAllowedRequiresSubscriptionWhenPolarConfigured(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")

	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":          models.BillingPlanNone,
			"plan_source":   models.BillingPlanSourcePolar,
			"trial_ends_at": time.Now().Add(-time.Hour),
		}).Error)

	err := models.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	require.ErrorIs(t, err, models.ErrHostedSubscriptionRequired)
}

func Test__ResolveOrganizationBillingPlanLeavesOpenTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	before, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, before)
	require.Equal(t, models.BillingPlanTrial, before.Plan)

	plan, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
}

func Test__ResolveOrganizationBillingPlanLapsesExpiredTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	expireOrganizationTrial(t, db, r.Organization.ID)

	plan, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.AllowsHostedExecution(time.Now()))
	assert.False(t, plan.AllowsCreditPurchase())

	stored, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, stored.Plan)
}

func Test__ResolveOrganizationBillingPlanInsertsNoneWhenMissing(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBillingPlan{}).Error)

	plan, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceSystem, plan.PlanSource)
	assert.False(t, plan.AllowsHostedExecution(time.Now()))

	stored, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, models.BillingPlanNone, stored.Plan)
}

func Test__ResolveOrganizationBillingPlanLeavesActiveBusiness(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)

	plan, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.True(t, plan.IsActiveBusiness())
	assert.True(t, plan.AllowsCreditPurchase())
}

func Test__ResolveOrganizationBillingPlanExpiresIncludedUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness)
	require.NoError(t, err)
	expireOrganizationTrial(t, db, r.Organization.ID)

	plan, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
}

func Test__AssertHostedRunAllowedRejectsExpiredTrialWithTopup(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	expireOrganizationTrial(t, db, r.Organization.ID)

	_, err := models.AddTopupLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(2500),
		uuid.NewString(),
	)
	require.NoError(t, err)

	err = models.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	require.ErrorIs(t, err, models.ErrHostedSubscriptionRequired)

	stored, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, stored.Plan)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Greater(t, summary.PurchasedRemainingMicros, int64(0))
}

func expireOrganizationTrial(t *testing.T, db *gorm.DB, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", orgID).
		Updates(map[string]any{
			"plan":          models.BillingPlanTrial,
			"trial_ends_at": time.Now().Add(-time.Hour),
		}).Error)
}
