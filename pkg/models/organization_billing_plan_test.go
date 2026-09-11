package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
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
