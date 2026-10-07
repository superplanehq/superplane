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

func Test__ApplyPolarSubscriptionReplacesAdminTrialWithPaid(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, nil)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)

	now := time.Now()
	end := now.AddDate(0, 1, 0)
	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_admin_trial",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
	)
	require.NoError(t, err)
	assert.True(t, grantIncluded)
	assert.Equal(t, models.BillingPlanBusiness, after.Plan)
	assert.Equal(t, models.BillingPlanSourcePolar, after.PlanSource)
	assert.True(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionReplacesAdminBusinessOnCancel(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
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
		models.PolarSubscriptionApply{
			ID:          "sub_admin",
			Status:      models.PolarSubscriptionStatusCanceled,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanTrial, after.Plan)
	assert.Equal(t, models.BillingPlanSourcePolar, after.PlanSource)
	assert.True(t, after.IsOpenTrial(time.Now()))
	assert.False(t, after.IsActiveBusiness())
}

func Test__SetAdminOrganizationPlanAllowsPolarCustomerWithoutSubscription(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	require.NoError(t, models.SetOrganizationPolarCustomerID(db, r.Organization.ID, "cust_polar"))

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__SetAdminOrganizationPlanRejectsActivePolarSubscription(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	now := time.Now()
	end := now.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{
		ID:          "sub_paid",
		Status:      models.PolarSubscriptionStatusActive,
		PeriodStart: &now,
		PeriodEnd:   &end,
	})
	require.NoError(t, err)

	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, nil)
	require.ErrorIs(t, err, models.ErrPolarManagedBillingPlan)

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourcePolar, plan.PlanSource)
}

func Test__SetAdminOrganizationPlanAllowsEndedPolarSubscription(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	now := time.Now()
	end := now.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{
		ID:          "sub_ended",
		Status:      models.PolarSubscriptionStatusActive,
		PeriodStart: &now,
		PeriodEnd:   &end,
	})
	require.NoError(t, err)
	ended := now.Add(-time.Hour)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"trial_ends_at":      ended,
			"current_period_end": ended,
		}).Error)
	_, _, err = models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{
		ID:                "sub_ended",
		Status:            models.PolarSubscriptionStatusCanceled,
		PeriodStart:       &now,
		PeriodEnd:         &ended,
		CancelAtPeriodEnd: true,
	})
	require.NoError(t, err)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, plan.Plan)
	assert.Equal(t, models.BillingPlanSourceAdmin, plan.PlanSource)
}

func Test__OrganizationBillingIsPolarManaged(t *testing.T) {
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	end := time.Now().Add(time.Hour)
	ended := time.Now().Add(-time.Hour)
	paid := &models.OrganizationBillingPlan{PolarSubscriptionStatus: models.PolarSubscriptionStatusActive}
	canceling := &models.OrganizationBillingPlan{
		PolarSubscriptionStatus: models.PolarSubscriptionStatusCanceled,
		CancelAtPeriodEnd:       true,
		CurrentPeriodEnd:        &end,
	}
	endedPlan := &models.OrganizationBillingPlan{
		PlanSource:              models.BillingPlanSourcePolar,
		PolarSubscriptionStatus: models.PolarSubscriptionStatusCanceled,
		CancelAtPeriodEnd:       true,
		CurrentPeriodEnd:        &ended,
	}
	incomplete := &models.OrganizationBillingPlan{
		PlanSource:              models.BillingPlanSourcePolar,
		PolarSubscriptionStatus: models.PolarSubscriptionStatusIncomplete,
	}
	adminTrial := &models.OrganizationBillingPlan{PlanSource: models.BillingPlanSourceAdmin}

	assert.True(t, models.OrganizationBillingIsPolarManaged(paid))
	assert.True(t, models.OrganizationBillingIsPolarManaged(canceling))
	assert.False(t, models.OrganizationBillingIsPolarManaged(endedPlan))
	assert.False(t, models.OrganizationBillingIsPolarManaged(incomplete))
	assert.False(t, models.OrganizationBillingIsPolarManaged(adminTrial))
	assert.False(t, models.OrganizationBillingIsPolarManaged(nil))

	t.Setenv("POLAR_ACCESS_TOKEN", "")
	assert.False(t, models.OrganizationBillingIsPolarManaged(paid))
}

func Test__SetAdminOrganizationPlanBusinessGrantsIncludedUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
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

	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
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

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
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

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanNone, nil)
	require.NoError(t, err)
	_, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanNone, nil)
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

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
	require.NoError(t, err)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, nil)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	assert.True(t, plan.IsOpenTrial(time.Now()))
	assert.False(t, plan.IsActiveBusiness())

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)
}

func Test__SetAdminOrganizationPlanExplicitTrialEndKeepsOpenStart(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	before, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, before.TrialStartedAt)
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	adminGrant, err := models.AddAdminLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(1100), "ops", nil)
	require.NoError(t, err)
	includedEnd := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	_, err = models.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(500),
		"order-included-"+r.Organization.ID.String(),
		includedEnd,
	)
	require.NoError(t, err)
	purchased, err := models.AddTopupLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(2500),
		"order-topup-"+r.Organization.ID.String(),
	)
	require.NoError(t, err)
	adminBefore := reloadGrant(t, db, adminGrant.ID)
	purchasedBefore := reloadGrant(t, db, purchased.ID)

	trialEnd := time.Now().Add(40 * 24 * time.Hour).UTC().Truncate(time.Second)
	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, &trialEnd)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanTrial, plan.Plan)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(trialEnd))
	require.NotNil(t, plan.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.Equal(*before.TrialStartedAt))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	assert.Equal(t, int64(1), countGrants(t, db, r.Organization.ID, models.LLMCreditGrantKindWelcome))
	assert.Equal(t, welcomeBefore.AmountMicros, welcome.AmountMicros)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(trialEnd))

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, welcome.AmountMicros, summary.WelcomeRemainingMicros)

	shorterEnd := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	plan, err = models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, &shorterEnd)
	require.NoError(t, err)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(shorterEnd))
	require.NotNil(t, plan.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.Equal(*before.TrialStartedAt))
	welcome = requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(shorterEnd))
	assert.Equal(t, welcomeBefore.AmountMicros, welcome.AmountMicros)

	assertGrantUnchanged(t, db, adminBefore)
	assertGrantUnchanged(t, db, purchasedBefore)
	assert.Equal(t, models.CentsToMicros(500), reloadIncludedAmount(t, db, r.Organization.ID))
}

func Test__SetAdminOrganizationPlanReopensExpiredTrialCredit(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	before, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, before.TrialStartedAt)
	ended := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Update("trial_ends_at", ended).Error)
	require.NoError(t, db.Model(&models.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", r.Organization.ID, models.LLMCreditGrantKindWelcome).
		Update("expires_at", ended).Error)

	trialEnd := time.Now().Add(21 * 24 * time.Hour).UTC().Truncate(time.Second)
	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, &trialEnd)
	require.NoError(t, err)
	require.NotNil(t, plan.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.After(*before.TrialStartedAt))
	assert.WithinDuration(t, time.Now(), *plan.TrialStartedAt, 5*time.Second)
	require.NotNil(t, plan.TrialEndsAt)
	assert.True(t, plan.TrialEndsAt.Equal(trialEnd))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	assert.Equal(t, int64(1), countGrants(t, db, r.Organization.ID, models.LLMCreditGrantKindWelcome))
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(trialEnd))

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, welcome.AmountMicros, summary.WelcomeRemainingMicros)
}

func Test__SetAdminOrganizationPlanTrialDoesNotCreateWelcomeGrant(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, models.LLMCreditGrantKindWelcome).
		Delete(&models.OrganizationLLMCreditGrant{}).Error)

	trialEnd := time.Now().Add(10 * 24 * time.Hour).UTC().Truncate(time.Second)
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, &trialEnd)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countGrants(t, db, r.Organization.ID, models.LLMCreditGrantKindWelcome))
}

func Test__SetAdminOrganizationPlanNilTrialEndKeepsFourteenDayDefault(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	before, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)

	plan, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanTrial, nil)
	require.NoError(t, err)
	require.NotNil(t, plan.TrialEndsAt)
	assert.WithinDuration(t, time.Now().Add(models.DefaultWelcomeGrantTTL), *plan.TrialEndsAt, 5*time.Second)
	require.NotNil(t, plan.TrialStartedAt)
	require.NotNil(t, before.TrialStartedAt)
	assert.True(t, plan.TrialStartedAt.Equal(*before.TrialStartedAt))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	require.NotNil(t, welcomeBefore.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(*welcomeBefore.ExpiresAt))
}

func Test__SetAdminOrganizationPlanBusinessAndNoneKeepWelcomeExpiry(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	planBefore, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	ignoredEnd := time.Now().Add(60 * 24 * time.Hour).UTC().Truncate(time.Second)

	business, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, &ignoredEnd)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanBusiness, business.Plan)
	require.NotNil(t, business.TrialEndsAt)
	require.NotNil(t, planBefore.TrialEndsAt)
	assert.True(t, business.TrialEndsAt.Equal(*planBefore.TrialEndsAt))
	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(*welcomeBefore.ExpiresAt))

	none, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanNone, &ignoredEnd)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, none.Plan)
	require.NotNil(t, none.TrialEndsAt)
	assert.True(t, none.TrialEndsAt.Equal(*planBefore.TrialEndsAt))
	welcome = requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(*welcomeBefore.ExpiresAt))
}

func requireWelcomeGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) models.OrganizationLLMCreditGrant {
	t.Helper()
	var grant models.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", orgID, models.LLMCreditGrantKindWelcome).First(&grant).Error)
	return grant
}

func countGrants(t *testing.T, db *gorm.DB, orgID uuid.UUID, kind string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", orgID, kind).
		Count(&count).Error)
	return count
}

func reloadGrant(t *testing.T, db *gorm.DB, id uuid.UUID) models.OrganizationLLMCreditGrant {
	t.Helper()
	var grant models.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("id = ?", id).First(&grant).Error)
	return grant
}

func reloadIncludedAmount(t *testing.T, db *gorm.DB, orgID uuid.UUID) int64 {
	t.Helper()
	var grant models.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", orgID, models.LLMCreditGrantKindIncluded).First(&grant).Error)
	return grant.AmountMicros
}

func assertGrantUnchanged(t *testing.T, db *gorm.DB, before models.OrganizationLLMCreditGrant) {
	t.Helper()
	var after models.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("id = ?", before.ID).First(&after).Error)
	assert.Equal(t, before.Kind, after.Kind)
	assert.Equal(t, before.AmountMicros, after.AmountMicros)
	if before.ExpiresAt == nil {
		assert.Nil(t, after.ExpiresAt)
		return
	}
	require.NotNil(t, after.ExpiresAt)
	assert.True(t, after.ExpiresAt.Equal(*before.ExpiresAt))
}

func Test__ApplyPolarSubscriptionCancelRestoresOpenTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_restore",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
	)
	require.NoError(t, err)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_restore",
			Status:      models.PolarSubscriptionStatusCanceled,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
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
		models.PolarSubscriptionApply{
			ID:          "sub_lapsed",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
	)
	require.NoError(t, err)

	ended := now.Add(-time.Hour)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Update("trial_ends_at", ended).Error)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_lapsed",
			Status:      models.PolarSubscriptionStatusCanceled,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
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
		models.PolarSubscriptionApply{
			ID:          "sub_incomplete",
			Status:      models.PolarSubscriptionStatusIncomplete,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
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

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
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

	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, models.BillingPlanBusiness, nil)
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

func Test__ApplyPolarSubscriptionCancelAtPeriodEndKeepsBusiness(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:                "sub_ending",
			Status:            models.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
	)
	require.NoError(t, err)
	assert.True(t, grantIncluded)
	assert.Equal(t, models.BillingPlanBusiness, after.Plan)
	assert.True(t, after.CancelAtPeriodEnd)
	assert.True(t, after.IsActiveBusiness())
	assert.True(t, after.AllowsCreditPurchase())
}

func Test__ApplyPolarSubscriptionCanceledKeepsBusinessUntilPeriodEnd(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_cancel_flag",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
	)
	require.NoError(t, err)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:                "sub_cancel_flag",
			Status:            models.PolarSubscriptionStatusCanceled,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanBusiness, after.Plan)
	assert.True(t, after.CancelAtPeriodEnd)
	assert.True(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionResumeClearsCancelAtPeriodEnd(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:                "sub_keep",
			Status:            models.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
	)
	require.NoError(t, err)

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_keep",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		},
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.Equal(t, models.BillingPlanBusiness, after.Plan)
	assert.False(t, after.CancelAtPeriodEnd)
	assert.True(t, after.IsActiveBusiness())
}

func Test__ApplyPolarSubscriptionIgnoresStaleSnapshot(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	older := now.Add(-time.Minute)
	newer := now

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_stale",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &older,
		},
	)
	require.NoError(t, err)

	resumed, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_stale",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &newer,
		},
	)
	require.NoError(t, err)
	require.False(t, resumed.CancelAtPeriodEnd)
	require.NotNil(t, resumed.PolarModifiedAt)
	assert.True(t, resumed.PolarModifiedAt.Equal(newer))

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:                "sub_stale",
			Status:            models.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
			ModifiedAt:        &older,
		},
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.False(t, after.CancelAtPeriodEnd)
	assert.Equal(t, models.BillingPlanBusiness, after.Plan)
	require.NotNil(t, after.PolarModifiedAt)
	assert.True(t, after.PolarModifiedAt.Equal(newer))
}

func Test__ApplyPolarSubscriptionTimestampLessSnapshotKeepsProviderOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now().UTC().Truncate(time.Second)
	end := now.AddDate(0, 1, 0)
	provider := now.Add(-30 * time.Second)
	laterWebhook := now.Add(-10 * time.Second)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_order",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &provider,
		},
	)
	require.NoError(t, err)

	canceled, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:                "sub_order",
			Status:            models.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
	)
	require.NoError(t, err)
	assert.True(t, canceled.CancelAtPeriodEnd)
	require.NotNil(t, canceled.PolarModifiedAt)
	assert.True(t, canceled.PolarModifiedAt.Equal(provider))

	after, grantIncluded, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:          "sub_order",
			Status:      models.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
			ModifiedAt:  &laterWebhook,
		},
	)
	require.NoError(t, err)
	assert.False(t, grantIncluded)
	assert.False(t, after.CancelAtPeriodEnd)
	require.NotNil(t, after.PolarModifiedAt)
	assert.True(t, after.PolarModifiedAt.Equal(laterWebhook))
}

func Test__ResolveOrganizationBillingPlanLapsesEndedCancelAtPeriodEnd(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	now := time.Now()
	end := now.AddDate(0, 1, 0)

	_, _, err := models.ApplyPolarSubscription(
		db,
		r.Organization.ID,
		models.PolarSubscriptionApply{
			ID:                "sub_ended",
			Status:            models.PolarSubscriptionStatusActive,
			PeriodStart:       &now,
			PeriodEnd:         &end,
			CancelAtPeriodEnd: true,
		},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"current_period_end": now.Add(-time.Hour),
			"trial_ends_at":      now.Add(-2 * time.Hour),
		}).Error)

	plan, err := models.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, plan.Plan)
	assert.False(t, plan.IsActiveBusiness())
	assert.False(t, plan.AllowsCreditPurchase())

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
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
