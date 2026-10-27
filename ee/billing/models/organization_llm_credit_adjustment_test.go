package models_test

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

func Test__AdjustCreditBalanceSetsTrialRemainingAfterSpend(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))

	before := describeCredit(t, db, r.Organization.ID)
	require.Greater(t, before.BilledMicros, int64(0))

	lowered, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   billingmodels.CentsToMicros(1000),
		ExpectedMicros: before.WelcomeRemainingMicros,
		Note:           "support",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, lowered)
	assert.Equal(t, billingmodels.LLMCreditGrantKindTrialAdjustment, lowered.Kind)
	assert.Equal(t, billingmodels.CentsToMicros(1000)-before.WelcomeRemainingMicros, lowered.AmountMicros)
	require.NotNil(t, lowered.ExpiresAt)
	assert.Equal(t, billingmodels.CentsToMicros(1000), describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)

	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   billingmodels.CentsToMicros(6000),
		ExpectedMicros: billingmodels.CentsToMicros(1000),
	})
	require.NoError(t, err)

	after := describeCredit(t, db, r.Organization.ID)
	assert.Equal(t, billingmodels.CentsToMicros(6000), after.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(6000), after.RemainingMicros)
}

func Test__AdjustCreditBalanceLoweredTrialSpillsToGrant(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	_, err := billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(10000), "", nil)
	require.NoError(t, err)

	before := describeCredit(t, db, r.Organization.ID)
	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   0,
		ExpectedMicros: before.WelcomeRemainingMicros,
	})
	require.NoError(t, err)

	execution := dispatchWorkOrderExecution(t, r)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))

	after := describeCredit(t, db, r.Organization.ID)
	require.Greater(t, after.BilledMicros, int64(0))
	assert.Equal(t, int64(0), after.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(10000)-after.BilledMicros, after.AdminRemainingMicros)
}

func Test__AdjustCreditBalanceTopupAndGrant(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	_, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTopup,
		TargetMicros:   billingmodels.CentsToMicros(2500),
		ExpectedMicros: billingmodels.CentsToMicros(10000),
	})
	require.NoError(t, err)
	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketGrant,
		TargetMicros:   billingmodels.CentsToMicros(700),
		ExpectedMicros: 0,
	})
	require.NoError(t, err)

	summary := describeCredit(t, db, r.Organization.ID)
	assert.Equal(t, billingmodels.CentsToMicros(2500), summary.PurchasedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(2500), summary.PurchasedCreditMicros)
	assert.Equal(t, billingmodels.CentsToMicros(700), summary.AdminRemainingMicros)

	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	kinds := make([]string, 0, len(grants))
	for _, grant := range grants {
		kinds = append(kinds, grant.Kind)
	}
	assert.Contains(t, kinds, billingmodels.LLMCreditGrantKindTopupAdjustment)
	assert.Contains(t, kinds, billingmodels.LLMCreditGrantKindAdminAdjustment)
}

func Test__AdjustCreditBalanceRejectsStaleExpectedBalance(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())

	_, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketGrant,
		TargetMicros:   billingmodels.CentsToMicros(500),
		ExpectedMicros: billingmodels.CentsToMicros(100),
	})
	require.ErrorIs(t, err, billingmodels.ErrCreditBalanceChanged)
	assert.Equal(t, int64(0), describeCredit(t, db, r.Organization.ID).AdminRemainingMicros)
}

func Test__AdjustCreditBalanceRejectsInvalidInput(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())

	_, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketGrant,
		TargetMicros:   -1,
	})
	require.ErrorIs(t, err, billingmodels.ErrCreditBalanceNegative)

	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         "invalid",
		TargetMicros:   billingmodels.CentsToMicros(100),
	})
	require.ErrorIs(t, err, billingmodels.ErrCreditBalanceBucket)
}

func Test__AdjustCreditBalanceRejectsTrialWithoutLiveWelcome(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	expireWelcomeGrant(t, db, r.Organization.ID)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{"plan": billingmodels.BillingPlanNone, "plan_id": db.Model(&billingmodels.BillingPlan{}).Select("id").Where("key = ?", "none")}).Error)

	_, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   billingmodels.CentsToMicros(1000),
		ExpectedMicros: 0,
	})
	require.ErrorIs(t, err, billingmodels.ErrTrialCreditNotActive)
}

func Test__AdjustCreditBalanceSetsTrialWhenWelcomeGrantIsMissingOnOpenTrial(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
	require.NoError(t, db.Model(&models.Account{}).
		Where("id = ?", r.Account.ID).
		Update("welcome_credit_granted_at", nil).Error)

	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.True(t, plan.IsOpenTrial(time.Now()))
	require.NotNil(t, plan.TrialEndsAt)

	target := billingmodels.CentsToMicros(1500)
	grant, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   target,
		ExpectedMicros: 0,
		Note:           "restore trial",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, grant)
	assert.Equal(t, billingmodels.LLMCreditGrantKindWelcome, grant.Kind)
	assert.Equal(t, target, grant.AmountMicros)
	assert.Equal(t, "restore trial", grant.Note)
	require.NotNil(t, grant.ActorAccountID)
	assert.Equal(t, r.Account.ID, *grant.ActorAccountID)
	require.NotNil(t, grant.ExpiresAt)
	assert.True(t, grant.ExpiresAt.Equal(plan.TrialEndsAt.UTC()))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(plan.TrialEndsAt.UTC()))
	assert.Equal(t, int64(0), countGrants(t, db, r.Organization.ID, billingmodels.LLMCreditGrantKindTrialAdjustment))
	assert.Equal(t, target, describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)

	var account models.Account
	require.NoError(t, db.Where("id = ?", r.Account.ID).First(&account).Error)
	assert.False(t, account.HasReceivedWelcomeCredit())
}

func Test__AdjustCreditBalanceSetsMissingTrialAfterHostedUsage(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)

	adminCents := int64(20000)
	_, err := billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(adminCents), "", nil)
	require.NoError(t, err)

	execution := dispatchWorkOrderExecution(t, r)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     20_000_000,
		TotalTokens:     20_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	before := describeCredit(t, db, r.Organization.ID)
	require.Greater(t, before.BilledMicros, billingmodels.CentsToMicros(5000))
	require.Less(t, before.BilledMicros, billingmodels.CentsToMicros(adminCents))
	assert.Equal(t, int64(0), before.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(adminCents)-before.BilledMicros, before.AdminRemainingMicros)
	assert.Equal(t, int64(0), before.PurchasedRemainingMicros)

	target := billingmodels.CentsToMicros(5000)
	grant, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   target,
		ExpectedMicros: 0,
		Note:           "restore trial",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, grant)
	assert.Equal(t, billingmodels.LLMCreditGrantKindWelcome, grant.Kind)
	assert.Equal(t, target+before.BilledMicros, grant.AmountMicros)

	after := describeCredit(t, db, r.Organization.ID)
	assert.Equal(t, target, after.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(adminCents), after.AdminRemainingMicros)
	assert.Equal(t, int64(0), after.PurchasedRemainingMicros)
	assert.Equal(t, int64(0), countGrants(t, db, r.Organization.ID, billingmodels.LLMCreditGrantKindTrialAdjustment))
}

func Test__AdjustCreditBalanceRaisesExpiredTrialOnOpenTrial(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	expireWelcomeGrant(t, db, r.Organization.ID)

	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.True(t, plan.IsOpenTrial(time.Now()))
	require.NotNil(t, plan.TrialEndsAt)
	require.Equal(t, int64(0), describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)

	target := billingmodels.CentsToMicros(1500)
	adjustment, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   target,
		ExpectedMicros: 0,
		Note:           "raise expired trial",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, adjustment)
	assert.Equal(t, billingmodels.LLMCreditGrantKindTrialAdjustment, adjustment.Kind)
	assert.Equal(t, target-welcomeBefore.AmountMicros, adjustment.AmountMicros)
	require.NotNil(t, adjustment.ExpiresAt)
	assert.True(t, adjustment.ExpiresAt.Equal(plan.TrialEndsAt.UTC()))

	welcome := requireWelcomeGrant(t, db, r.Organization.ID)
	assert.Equal(t, welcomeBefore.AmountMicros, welcome.AmountMicros)
	require.NotNil(t, welcome.ExpiresAt)
	assert.True(t, welcome.ExpiresAt.Equal(plan.TrialEndsAt.UTC()))
	assert.Equal(t, target, describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)
	assert.Equal(t, int64(1), countGrants(t, db, r.Organization.ID, billingmodels.LLMCreditGrantKindTrialAdjustment))
}

func Test__AdjustCreditBalanceRaisesExpiredTrialAfterHostedUsage(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	welcomeBefore := requireWelcomeGrant(t, db, r.Organization.ID)
	expireWelcomeGrant(t, db, r.Organization.ID)

	adminCents := int64(20000)
	_, err := billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(adminCents), "", nil)
	require.NoError(t, err)

	execution := dispatchWorkOrderExecution(t, r)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     20_000_000,
		TotalTokens:     20_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	before := describeCredit(t, db, r.Organization.ID)
	require.Greater(t, before.BilledMicros, welcomeBefore.AmountMicros)
	require.Less(t, before.BilledMicros, billingmodels.CentsToMicros(adminCents))
	assert.Equal(t, int64(0), before.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(adminCents)-before.BilledMicros, before.AdminRemainingMicros)

	target := billingmodels.CentsToMicros(5000)
	adjustment, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   target,
		ExpectedMicros: 0,
	})
	require.NoError(t, err)
	require.NotNil(t, adjustment)
	assert.Equal(t, billingmodels.LLMCreditGrantKindTrialAdjustment, adjustment.Kind)
	assert.Equal(t, target+before.BilledMicros-welcomeBefore.AmountMicros, adjustment.AmountMicros)

	after := describeCredit(t, db, r.Organization.ID)
	assert.Equal(t, target, after.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(adminCents), after.AdminRemainingMicros)
}

func Test__AdjustCreditBalanceSkipsUnchangedBalance(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	before := describeCredit(t, db, r.Organization.ID)

	grant, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   before.WelcomeRemainingMicros,
		ExpectedMicros: before.WelcomeRemainingMicros,
	})
	require.NoError(t, err)
	assert.Nil(t, grant)

	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Len(t, grants, 1)
}

func describeCredit(t *testing.T, db *gorm.DB, orgID uuid.UUID) billingmodels.OrganizationLLMCreditSummary {
	t.Helper()
	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, orgID)
	require.NoError(t, err)
	return summary
}

func TestIncludedBalanceAfterExhaustionAndExpiration(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()
	// Spend before the first included grant so a simple target-current delta would be insufficient.
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).Where("organization_id = ?", r.Organization.ID).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	execution := dispatchWorkOrderExecution(t, r)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))
	before := describeCredit(t, db, r.Organization.ID)
	require.Positive(t, before.BilledMicros)
	adjustment := billingmodels.CreditBalanceAdjustment{OrganizationID: r.Organization.ID, Bucket: billingmodels.CreditBalanceBucketIncluded, TargetMicros: billingmodels.CentsToMicros(300), ExpectedMicros: 0, ActorAccountID: &r.Account.ID, Note: "Support correction"}
	_, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, adjustment)
	require.ErrorIs(t, err, billingmodels.ErrCreditExpiration)
	choice := "at"
	expiry := time.Now().Add(time.Hour)
	adjustment.ExpirationChoice = &choice
	adjustment.ExpiresAt = &expiry
	grant, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, adjustment)
	require.NoError(t, err)
	require.Equal(t, billingmodels.LLMCreditGrantKindIncludedAdjustment, grant.Kind)
	require.Equal(t, &r.Account.ID, grant.ActorAccountID)
	require.Equal(t, "Support correction", grant.Note)
	require.Equal(t, billingmodels.CentsToMicros(300), describeCredit(t, db, r.Organization.ID).IncludedRemainingMicros)
	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, adjustment)
	require.ErrorIs(t, err, billingmodels.ErrCreditBalanceChanged)
	adjustment.ExpectedMicros = billingmodels.CentsToMicros(300)
	adjustment.TargetMicros = billingmodels.CentsToMicros(100)
	reduction, err := billingmodels.AdjustOrganizationLLMCreditBalance(db, adjustment)
	require.NoError(t, err)
	require.Negative(t, reduction.AmountMicros)
	require.Equal(t, billingmodels.CentsToMicros(100), describeCredit(t, db, r.Organization.ID).IncludedRemainingMicros)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).Where("id = ?", grant.ID).Update("expires_at", time.Now().Add(-time.Minute)).Error)
	require.Zero(t, describeCredit(t, db, r.Organization.ID).IncludedRemainingMicros)
	adjustment.ExpectedMicros = 0
	adjustment.TargetMicros = billingmodels.CentsToMicros(50)
	choice = "never"
	adjustment.ExpiresAt = nil
	grant, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, adjustment)
	require.NoError(t, err)
	require.Nil(t, grant.ExpiresAt)
	require.Equal(t, billingmodels.CentsToMicros(50), describeCredit(t, db, r.Organization.ID).IncludedRemainingMicros)
	plan, err := billingmodels.ResolveOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.False(t, plan.AllowsCreditPurchase())
	require.NoError(t, billingmodels.ExpireOpenIncludedGrants(db, r.Organization.ID))
	require.Zero(t, describeCredit(t, db, r.Organization.ID).IncludedRemainingMicros)
}
