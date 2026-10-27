package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__ApplyMarkupMicros(t *testing.T) {
	assert.Equal(t, int64(0), billingmodels.ApplyMarkupMicros(0, 2000))
	assert.Equal(t, int64(100), billingmodels.ApplyMarkupMicros(100, 0))
	assert.Equal(t, int64(120), billingmodels.ApplyMarkupMicros(100, 2000))
}

func Test__WelcomeGrantOnOrgCreate(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.GrantMicros)
	assert.Equal(t, summary.GrantMicros, summary.RemainingMicros)
	assert.False(t, summary.Warning)

	require.NoError(t, billingmodels.GrantWelcomeCredit(db, r.Organization.ID, r.Account.ID))
	again, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, summary.GrantMicros, again.GrantMicros)

	account, err := models.FindAccountByID(r.Account.ID.String())
	require.NoError(t, err)
	assert.True(t, account.HasReceivedWelcomeCredit())
}

func Test__WelcomeGrantSetsExpiresAt(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	var grant billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
		First(&grant).Error)
	require.NotNil(t, grant.ExpiresAt)
	assert.WithinDuration(t, grant.CreatedAt.AddDate(0, 0, billingmodels.SignupCreditDays), *grant.ExpiresAt, time.Second)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, summary.WelcomeCreditExpiresAt)
	assert.WithinDuration(t, *grant.ExpiresAt, *summary.WelcomeCreditExpiresAt, time.Second)
}

func Test__SignupCreditIsOncePerAccount(t *testing.T) {
	restoreInstallationLLMSettings(t)
	support.Setup(t)
	db := database.DB(t.Context())
	account, err := models.CreateAccountInTransaction(db, support.RandomName("account"), support.RandomName("owner")+"@example.com")
	require.NoError(t, err)
	organization, err := models.CreateOrganizationInTransaction(db, support.RandomName("org"), support.RandomName("org"))
	require.NoError(t, err)
	require.NoError(t, billingmodels.GrantWelcomeCredit(db, organization.ID, account.ID))
	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, organization.ID)
	require.NoError(t, err)
	require.Equal(t, billingmodels.CentsToMicros(1000), summary.GrantMicros)
	require.NotNil(t, summary.WelcomeCreditExpiresAt)
	require.WithinDuration(t, time.Now().AddDate(0, 0, 365), *summary.WelcomeCreditExpiresAt, time.Minute)
	require.NoError(t, billingmodels.GrantWelcomeCredit(db, organization.ID, account.ID))
	again, err := billingmodels.DescribeOrganizationLLMCredit(db, organization.ID)
	require.NoError(t, err)
	require.Equal(t, summary.GrantMicros, again.GrantMicros)
	second, err := models.CreateOrganizationInTransaction(db, support.RandomName("org"), support.RandomName("org"))
	require.NoError(t, err)
	require.NoError(t, billingmodels.GrantWelcomeCredit(db, second.ID, account.ID))
	secondCredit, err := billingmodels.DescribeOrganizationLLMCredit(db, second.ID)
	require.NoError(t, err)
	require.Zero(t, secondCredit.GrantMicros)
}

func Test__ExpiredWelcomeCreditIsNotUsable(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()
	expireWelcomeGrant(t, db, r.Organization.ID)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.GrantMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, int64(0), summary.RemainingMicros)
	require.ErrorIs(t, billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID), billingmodels.ErrHostedCreditEmpty)
}

func Test__ExpiredWelcomeSpendDoesNotReducePurchasedCredit(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	before, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, before.BilledMicros, int64(0))
	require.Greater(t, before.RemainingMicros, int64(0))

	expiredAt := expireWelcomeGrant(t, db, r.Organization.ID)
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("organization_id = ? AND funding_source = ? AND usage_kind = ?", r.Organization.ID, models.UsageFundingSourceHosted, models.UsageKindModel).
		Update("occurred_at", expiredAt.Add(-time.Second)).Error)
	_, err = billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), after.PurchasedCreditMicros)
	assert.Equal(t, billingmodels.CentsToMicros(10000), after.RemainingMicros)
	assert.Equal(t, before.BilledMicros, after.BilledMicros)
	require.NoError(t, billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID))
}

func Test__ExpiredWelcomeSpendDoesNotReduceAdminGrant(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))

	before, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, before.BilledMicros, int64(0))
	require.Greater(t, before.RemainingMicros, int64(0))

	expiredAt := expireWelcomeGrant(t, db, r.Organization.ID)
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("organization_id = ? AND funding_source = ? AND usage_kind = ?", r.Organization.ID, models.UsageFundingSourceHosted, models.UsageKindModel).
		Update("occurred_at", expiredAt.Add(-time.Second)).Error)

	const adminCents = int64(2000)
	_, err = billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(adminCents), "support", &r.Account.ID)
	require.NoError(t, err)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	postExpirySpend := summary.BilledMicros - before.BilledMicros
	require.Greater(t, postExpirySpend, int64(0))
	expected := billingmodels.CentsToMicros(adminCents) - postExpirySpend
	require.Greater(t, expected, int64(0))
	assert.Equal(t, int64(0), summary.WelcomeRemainingMicros)
	assert.Equal(t, expected, summary.AdminRemainingMicros)
	assert.Equal(t, expected, summary.RemainingMicros)

	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	require.NoError(t, billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil))
}

func recordHostedPromptUsage(t *testing.T, db *gorm.DB, orgID, runID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  orgID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))
}

func Test__ExpiredUnusedWelcomeDoesNotShieldLaterPurchasedSpend(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	expireWelcomeGrant(t, db, r.Organization.ID)
	_, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	execution := dispatchWorkOrderExecution(t, r)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, summary.BilledMicros, int64(0))
	assert.Equal(t, billingmodels.CentsToMicros(10000), summary.PurchasedCreditMicros)
	assert.Equal(t, billingmodels.CentsToMicros(10000)-summary.BilledMicros, summary.RemainingMicros)
}

func expireWelcomeGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) time.Time {
	t.Helper()
	expired := time.Now().Add(-time.Minute)
	require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", orgID, billingmodels.LLMCreditGrantKindWelcome).
		Update("expires_at", expired).Error)
	return expired
}

func Test__WelcomeGrantOnlyOncePerAccount(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	second, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, billingmodels.GrantWelcomeCredit(db, second.ID, r.Account.ID))

	first, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), first.GrantMicros)

	skipped, err := billingmodels.DescribeOrganizationLLMCredit(db, second.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), skipped.GrantMicros)

	viaSupport := support.CreateOrganization(t, r, r.User)
	viaSupportSummary, err := billingmodels.DescribeOrganizationLLMCredit(db, viaSupport.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), viaSupportSummary.GrantMicros)

	other, err := models.CreateAccount("Other User", "other-welcome@example.com")
	require.NoError(t, err)
	third, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, billingmodels.GrantWelcomeCredit(db, third.ID, other.ID))

	granted, err := billingmodels.DescribeOrganizationLLMCredit(db, third.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), granted.GrantMicros)

	reloaded, err := models.FindAccountByID(other.ID.String())
	require.NoError(t, err)
	assert.True(t, reloaded.HasReceivedWelcomeCredit())
}

func Test__WelcomeGrantStampsAccountWhenOrgAlreadyHasWelcome(t *testing.T) {
	restoreInstallationLLMSettings(t)
	db := database.Conn()

	account, err := models.CreateAccount("Legacy Welcome", "legacy-welcome@example.com")
	require.NoError(t, err)
	org, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)

	now := time.Now()
	expiresAt := now.Add(billingmodels.DefaultWelcomeGrantTTL)
	require.NoError(t, db.Create(&billingmodels.OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		Kind:           billingmodels.LLMCreditGrantKindWelcome,
		AmountMicros:   billingmodels.CentsToMicros(billingmodels.SignupCreditCents),
		CreatedAt:      now,
		ExpiresAt:      &expiresAt,
	}).Error)

	require.NoError(t, billingmodels.GrantWelcomeCredit(db, org.ID, account.ID))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.GrantMicros)

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.True(t, reloaded.HasReceivedWelcomeCredit())
}

func Test__SignupCreditIgnoresTrialCatalogAmount(t *testing.T) {
	restoreInstallationLLMSettings(t)
	_ = support.Setup(t)
	db := database.Conn()

	err := db.Model(&billingmodels.BillingPlan{}).Where("key = ?", "trial").Update("trial_credit_cents", 0).Error
	require.NoError(t, err)

	account, err := models.CreateAccount("Catalog Zero", "catalog-zero@example.com")
	require.NoError(t, err)
	org, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, billingmodels.GrantWelcomeCredit(db, org.ID, account.ID))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.GrantMicros)
	plan, err := billingmodels.FindOrganizationBillingPlan(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	require.ErrorIs(t, billingmodels.AssertOrganizationPlanAllowsRuns(db, org.ID), billingmodels.ErrHostedSubscriptionRequired)

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.True(t, reloaded.HasReceivedWelcomeCredit())
}

func Test__AdminGrantRestoresHostedCredit(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	execution := dispatchWorkOrderExecution(t, r)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, summary.BilledMicros, int64(0))
	assert.Equal(t, billingmodels.ApplyMarkupMicros(300*billingmodels.MicrosPerCent, billingmodels.DefaultMarkupBPS), summary.BilledMicros)

	actor := r.Account.ID
	_, err = billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, summary.BilledMicros, "restore", &actor)
	require.NoError(t, err)

	restored, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, summary.GrantMicros+summary.BilledMicros, restored.GrantMicros)
	assert.Equal(t, summary.GrantMicros, restored.RemainingMicros)
}

func Test__HostedRecordUsageAppliesOrgMarkupOverride(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	override := 0
	require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(db, r.Organization.ID, &override))

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	var event models.WorkspaceUsageEvent
	require.NoError(t, db.Where("work_order_execution_id = ?", execution.ID).First(&event).Error)
	assert.Equal(t, models.UsageFundingSourceHosted, event.FundingSource)
	assert.Equal(t, int64(3_000_000), event.ProviderCostMicros)
	assert.Equal(t, int64(3_000_000), event.CostMicros)
}

func Test__BYOKRecordUsageIsNotMarkedUp(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))

	var event models.WorkspaceUsageEvent
	require.NoError(t, db.Where("work_order_execution_id = ?", execution.ID).First(&event).Error)
	assert.Equal(t, models.UsageFundingSourceBYOK, event.FundingSource)
	assert.Equal(t, event.ProviderCostMicros, event.CostMicros)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.BilledMicros)
}

func Test__DescribeOrganizationLLMCreditSubtractsComputeUsage(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	runID := requireExecutionRunID(t, execution)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))
	modelBilledMicros := billingmodels.ApplyMarkupMicros(300*billingmodels.MicrosPerCent, billingmodels.DefaultMarkupBPS)

	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 3600,
	}))
	var computeCostMicros int64
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Select("COALESCE(SUM(cost_micros), 0)").
		Where("canvas_run_id = ? AND usage_kind = ?", runID, models.UsageKindCompute).
		Scan(&computeCostMicros).Error)
	require.Greater(t, computeCostMicros, int64(0))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, modelBilledMicros+computeCostMicros, summary.BilledMicros)
	assert.Equal(t, summary.GrantMicros-summary.BilledMicros, summary.RemainingMicros)
}

func Test__ComputeUsageAloneCanExhaustHostedCredit(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	runID := requireExecutionRunID(t, execution)

	before, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, before.RemainingMicros, int64(0))

	// e1-large-amd64 at ~$2/hr; enough seconds to exceed the welcome grant.
	secondsToExhaust := before.RemainingMicros/pricebook.MicrosPerSecondE1Large + 10
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: secondsToExhaust,
	}))

	err = billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID)
	require.ErrorIs(t, err, billingmodels.ErrHostedCreditEmpty)

	err = billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	require.ErrorIs(t, err, billingmodels.ErrHostedCreditEmpty)
}

func Test__AssertHostedCreditAvailable(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	require.NoError(t, billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID))

	err := db.Model(&billingmodels.BillingPlan{}).Where("key = ?", "trial").Update("trial_credit_cents", 0).Error
	require.NoError(t, err)

	org, err := models.CreateOrganization(support.RandomName("empty-credit"), "")
	require.NoError(t, err)
	err = billingmodels.AssertHostedCreditAvailable(db, org.ID)
	require.ErrorIs(t, err, billingmodels.ErrHostedCreditEmpty)
}

func Test__AssertHostedRunAllowedAllowsConcurrentStarts(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.Conn()

	errs := make(chan error, 2)
	go func() {
		errs <- billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	}()
	go func() {
		errs <- billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	}()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
}

func Test__PolarGrantIsIdempotentByOrderID(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()

	first, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(2500), orderID)
	require.NoError(t, err)
	second, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(2500), orderID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents)+billingmodels.CentsToMicros(2500), summary.GrantMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, billingmodels.CentsToMicros(2500), summary.PurchasedCreditMicros)
}

func Test__DescribeOrganizationLLMCreditSplitsSuperPlaneAndPurchasedGrants(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	_, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, billingmodels.CentsToMicros(10000), summary.PurchasedCreditMicros)
	assert.Equal(t, summary.SuperPlaneGrantMicros+summary.PurchasedCreditMicros, summary.GrantMicros)
	assert.Equal(t, summary.GrantMicros, summary.RemainingMicros)
}

func Test__PolarRefundIsIdempotentAndCapsAtGrant(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	refundID := uuid.NewString()

	_, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(2500), orderID)
	require.NoError(t, err)

	first, err := billingmodels.AddPolarLLMCreditRefund(db, r.Organization.ID, billingmodels.CentsToMicros(1000), orderID, refundID)
	require.NoError(t, err)
	second, err := billingmodels.AddPolarLLMCreditRefund(db, r.Organization.ID, billingmodels.CentsToMicros(1000), orderID, refundID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(1000), reversed)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents)+billingmodels.CentsToMicros(1500), summary.GrantMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, billingmodels.CentsToMicros(1500), summary.PurchasedCreditMicros)
}

func Test__ReversePolarOrderCreditDoesNotOverDebitConcurrently(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	_, err := billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(2500), orderID)
	require.NoError(t, err)

	errs := make(chan error, 2)
	go func() {
		errs <- billingmodels.ReversePolarOrderCredit(db, r.Organization.ID, orderID, billingmodels.CentsToMicros(2500), orderID+":full")
	}()
	go func() {
		errs <- billingmodels.ReversePolarOrderCredit(db, r.Organization.ID, orderID, billingmodels.CentsToMicros(1000), orderID+":partial")
	}()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)

	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(2500), reversed)
}

func Test__FactoryHostedBudgetZeroBlocksHostedStart(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.Conn()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	zero := int64(0)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &zero))

	err = billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID)
	require.ErrorIs(t, err, billingmodels.ErrFactoryHostedBudgetEmpty)
}

func Test__FactoryHostedBudgetHardStopWhenSpent(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	workOrderExecution := dispatchWorkOrderExecution(t, r)
	factory, err := models.FindFactory(db, r.Organization.ID, workOrderExecution.FactoryID)
	require.NoError(t, err)
	budget := int64(1)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &budget))

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, workOrderExecution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	err = billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID)
	require.ErrorIs(t, err, billingmodels.ErrFactoryHostedBudgetEmpty)
}

func Test__FactoryHostedBudgetHardStopWhenSpentOnCompute(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.DB(t.Context())
	workOrderExecution := dispatchWorkOrderExecution(t, r)
	factory, err := models.FindFactory(db, r.Organization.ID, workOrderExecution.FactoryID)
	require.NoError(t, err)
	budget := int64(1)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &budget))

	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, workOrderExecution),
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 180,
	}))

	err = billingmodels.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID)
	require.ErrorIs(t, err, billingmodels.ErrFactoryHostedBudgetEmpty)
}

func restoreInstallationLLMSettings(t *testing.T) {
	t.Helper()
	resetInstallationLLMSettings(t)
	t.Cleanup(func() {
		resetInstallationLLMSettings(t)
	})
}

func resetInstallationLLMSettings(t *testing.T) {
	t.Helper()
	require.NoError(t, database.Conn().Model(&billingmodels.BillingPlan{}).Where("key = ?", "trial").Updates(map[string]any{"trial_credit_cents": billingmodels.DefaultWelcomeGrantCents, "trial_length_days": 14}).Error)
	_, err := billingmodels.UpdateInstallationLLMSettings(database.Conn(), billingmodels.InstallationLLMSettings{
		WelcomeGrantCents:   billingmodels.DefaultWelcomeGrantCents,
		MarkupBPS:           billingmodels.DefaultMarkupBPS,
		WarningThresholdBPS: billingmodels.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)
}

func Test__CentsToMicros(t *testing.T) {
	assert.Equal(t, int64(50_000_000), billingmodels.CentsToMicros(5000))
	assert.Equal(t, int64(0), billingmodels.CentsToMicros(0))
}

func Test__SignedMicrosToCents(t *testing.T) {
	assert.Equal(t, int64(2500), billingmodels.SignedMicrosToCents(billingmodels.CentsToMicros(2500)))
	assert.Equal(t, int64(-1000), billingmodels.SignedMicrosToCents(-billingmodels.CentsToMicros(1000)))
	assert.Equal(t, int64(0), billingmodels.SignedMicrosToCents(0))
}

func Test__ListOrganizationLLMCreditGrants(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	actor := r.Account.ID
	_, err := billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(1500), "Support grant", &actor)
	require.NoError(t, err)

	orderID := uuid.NewString()
	_, err = billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(2500), orderID)
	require.NoError(t, err)
	_, err = billingmodels.AddPolarLLMCreditRefund(db, r.Organization.ID, billingmodels.CentsToMicros(500), orderID, orderID+":partial")
	require.NoError(t, err)

	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(grants), 4)
	assert.Equal(t, billingmodels.LLMCreditGrantKindTopupRefund, grants[0].Kind)
	assert.Equal(t, billingmodels.LLMCreditGrantKindTopup, grants[1].Kind)
	assert.Equal(t, billingmodels.LLMCreditGrantKindAdmin, grants[2].Kind)
	assert.Equal(t, billingmodels.LLMCreditGrantKindWelcome, grants[3].Kind)

	names, err := billingmodels.CreditGrantActorNames(db, grants)
	require.NoError(t, err)
	assert.Equal(t, r.Account.Name, names[actor])
}

func Test__ExpireOpenIncludedGrantsLeavesTopupSpendable(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	_, err := billingmodels.AddTopupLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(2500),
		orderID,
	)
	require.NoError(t, err)

	periodEnd := time.Now().AddDate(0, 1, 0)
	includedKey := billingmodels.IncludedGrantKey("sub_expire", periodEnd)
	_, err = billingmodels.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents),
		includedKey,
		periodEnd,
	)
	require.NoError(t, err)

	require.NoError(t, billingmodels.ExpireOpenIncludedGrants(db, r.Organization.ID))
	require.NoError(t, billingmodels.ExpireOpenIncludedGrants(db, r.Organization.ID))

	grant := findCanceledIncludedGrant(t, db, r.Organization.ID)
	require.NotNil(t, grant.ExpiresAt)
	assert.False(t, grant.ExpiresAt.After(time.Now()))

	_, err = billingmodels.FindLLMCreditGrantByPolarOrderID(db, includedKey)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(2500), summary.PurchasedRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents)+billingmodels.CentsToMicros(2500), summary.RemainingMicros)
}

func Test__ExpireOpenIncludedGrantsAfterResubscribeSameKey(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodEnd := time.Now().AddDate(0, 1, 0)
	includedKey := billingmodels.IncludedGrantKey("sub_expire_again", periodEnd)

	_, err := billingmodels.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents),
		includedKey,
		periodEnd,
	)
	require.NoError(t, err)
	require.NoError(t, billingmodels.ExpireOpenIncludedGrants(db, r.Organization.ID))

	_, err = billingmodels.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		billingmodels.CentsToMicros(billingmodels.DefaultIncludedGrantCents),
		includedKey,
		periodEnd,
	)
	require.NoError(t, err)
	require.NoError(t, billingmodels.ExpireOpenIncludedGrants(db, r.Organization.ID))

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)

	_, err = billingmodels.FindLLMCreditGrantByPolarOrderID(db, includedKey)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func findCanceledIncludedGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) billingmodels.OrganizationLLMCreditGrant {
	t.Helper()
	grants, err := billingmodels.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	now := time.Now()
	for _, grant := range grants {
		if grant.Kind != billingmodels.LLMCreditGrantKindIncluded {
			continue
		}
		if !grant.IsExpired(now) {
			continue
		}
		require.NotNil(t, grant.PolarOrderID)
		return grant
	}
	require.FailNow(t, "canceled included grant not found")
	return billingmodels.OrganizationLLMCreditGrant{}
}
