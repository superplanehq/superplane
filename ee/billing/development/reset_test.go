package development

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestResetOrganizationBillingTrialRestoresTrial(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	catalog, err := billingmodels.FindBillingPlan(db, billingmodels.BillingPlanTrial)
	require.NoError(t, err)
	_, err = billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, 1_000_000, "local top-up", &r.Account.ID)
	require.NoError(t, err)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Update("expires_at", time.Now().Add(-time.Hour)).Error)
	require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(db, r.Organization.ID, "cus_local"))
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":                      billingmodels.BillingPlanBusiness,
			"plan_source":               billingmodels.BillingPlanSourcePolar,
			"polar_subscription_id":     "sub_local",
			"polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
		}).Error)
	require.NoError(t, db.Create(&models.WorkspaceUsageEvent{
		ID:               uuid.New(),
		UsageKind:        models.UsageKindModel,
		FundingSource:    models.UsageFundingSourceBYOK,
		Currency:         "usd",
		PriceBookVersion: "reset-test",
		IdempotencyKey:   uuid.NewString(),
		OccurredAt:       time.Now(),
		CreatedAt:        time.Now(),
		OrganizationID:   r.Organization.ID,
		CanvasRunID:      uuid.New(),
		NodeExecutionID:  uuid.New(),
		NodeID:           "prompt",
		Provider:         models.UsageProviderAnthropic,
		Model:            "claude-sonnet-4-6",
		InputTokens:      1_000_000,
		TotalTokens:      1_000_000,
	}).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ResetOrganizationBillingTrial(tx, r.Organization.ID)
	}))

	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceSystem, plan.PlanSource)
	assert.Nil(t, plan.PolarSubscriptionID)
	require.NotNil(t, plan.TrialEndsAt)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, catalog.TrialLengthDays), *plan.TrialEndsAt, 5*time.Second)
	settings, err := billingmodels.FindOrganizationLLMSettings(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	assert.Nil(t, settings.PolarCustomerID)
	var extraGrants, usageEvents int64
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind <> ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Count(&extraGrants).Error)
	assert.Zero(t, extraGrants)
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("organization_id = ?", r.Organization.ID).Count(&usageEvents).Error)
	assert.Zero(t, usageEvents)
	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.RemainingMicros)
	require.NotNil(t, summary.WelcomeCreditExpiresAt)
	assert.WithinDuration(t, *plan.TrialEndsAt, *summary.WelcomeCreditExpiresAt, time.Second)
}
