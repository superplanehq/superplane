package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ApplyMarkupMicros(t *testing.T) {
	assert.Equal(t, int64(0), models.ApplyMarkupMicros(0, 2000))
	assert.Equal(t, int64(100), models.ApplyMarkupMicros(100, 0))
	assert.Equal(t, int64(120), models.ApplyMarkupMicros(100, 2000))
}

func Test__WelcomeGrantOnOrgCreate(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.GrantMicros)
	assert.Equal(t, summary.GrantMicros, summary.RemainingMicros)
	assert.False(t, summary.Warning)

	require.NoError(t, models.GrantWelcomeCredit(db, r.Organization.ID, r.Account.ID))
	again, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
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

	var grant models.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, models.LLMCreditGrantKindWelcome).
		First(&grant).Error)
	require.NotNil(t, grant.ExpiresAt)
	assert.WithinDuration(t, grant.CreatedAt.Add(models.DefaultWelcomeGrantTTL), *grant.ExpiresAt, time.Second)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, summary.WelcomeCreditExpiresAt)
	assert.WithinDuration(t, *grant.ExpiresAt, *summary.WelcomeCreditExpiresAt, time.Second)
}

func Test__ExpiredWelcomeCreditIsNotUsable(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()
	expireWelcomeGrant(t, db, r.Organization.ID)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.GrantMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, int64(0), summary.RemainingMicros)
	require.ErrorIs(t, models.AssertHostedCreditAvailable(db, r.Organization.ID), models.ErrHostedCreditEmpty)
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

	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, before.BilledMicros, int64(0))
	require.Greater(t, before.RemainingMicros, int64(0))

	expiredAt := expireWelcomeGrant(t, db, r.Organization.ID)
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("organization_id = ? AND funding_source = ? AND usage_kind = ?", r.Organization.ID, models.UsageFundingSourceHosted, models.UsageKindModel).
		Update("occurred_at", expiredAt.Add(-time.Second)).Error)
	_, err = models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(10000), after.PurchasedCreditMicros)
	assert.Equal(t, models.CentsToMicros(10000), after.RemainingMicros)
	assert.Equal(t, before.BilledMicros, after.BilledMicros)
	require.NoError(t, models.AssertHostedCreditAvailable(db, r.Organization.ID))
}

func Test__ExpiredUnusedWelcomeDoesNotShieldLaterPurchasedSpend(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	expireWelcomeGrant(t, db, r.Organization.ID)
	_, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(10000), uuid.NewString())
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

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, summary.BilledMicros, int64(0))
	assert.Equal(t, models.CentsToMicros(10000), summary.PurchasedCreditMicros)
	assert.Equal(t, models.CentsToMicros(10000)-summary.BilledMicros, summary.RemainingMicros)
}

func expireWelcomeGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) time.Time {
	t.Helper()
	expired := time.Now().Add(-time.Minute)
	require.NoError(t, db.Model(&models.OrganizationLLMCreditGrant{}).
		Where("organization_id = ? AND kind = ?", orgID, models.LLMCreditGrantKindWelcome).
		Update("expires_at", expired).Error)
	return expired
}

func Test__WelcomeGrantOnlyOncePerAccount(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	second, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, models.GrantWelcomeCredit(db, second.ID, r.Account.ID))

	first, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), first.GrantMicros)

	skipped, err := models.DescribeOrganizationLLMCredit(db, second.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), skipped.GrantMicros)

	viaSupport := support.CreateOrganization(t, r, r.User)
	viaSupportSummary, err := models.DescribeOrganizationLLMCredit(db, viaSupport.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), viaSupportSummary.GrantMicros)

	other, err := models.CreateAccount("Other User", "other-welcome@example.com")
	require.NoError(t, err)
	third, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, models.GrantWelcomeCredit(db, third.ID, other.ID))

	granted, err := models.DescribeOrganizationLLMCredit(db, third.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), granted.GrantMicros)

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
	expiresAt := now.Add(models.DefaultWelcomeGrantTTL)
	require.NoError(t, db.Create(&models.OrganizationLLMCreditGrant{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		Kind:           models.LLMCreditGrantKindWelcome,
		AmountMicros:   models.CentsToMicros(models.DefaultWelcomeGrantCents),
		CreatedAt:      now,
		ExpiresAt:      &expiresAt,
	}).Error)

	require.NoError(t, models.GrantWelcomeCredit(db, org.ID, account.ID))

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.GrantMicros)

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.True(t, reloaded.HasReceivedWelcomeCredit())
}

func Test__WelcomeGrantSkippedWhenAmountIsZero(t *testing.T) {
	restoreInstallationLLMSettings(t)
	_ = support.Setup(t)
	db := database.Conn()

	_, err := models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		WelcomeGrantCents:   0,
		MarkupBPS:           models.DefaultMarkupBPS,
		WarningThresholdBPS: models.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)

	account, err := models.CreateAccount("Zero Welcome", "zero-welcome@example.com")
	require.NoError(t, err)
	org, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, models.GrantWelcomeCredit(db, org.ID, account.ID))

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.GrantMicros)

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.False(t, reloaded.HasReceivedWelcomeCredit())
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

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, summary.BilledMicros, int64(0))
	assert.Equal(t, models.ApplyMarkupMicros(300*models.MicrosPerCent, models.DefaultMarkupBPS), summary.BilledMicros)

	actor := r.Account.ID
	_, err = models.AddAdminLLMCreditGrant(db, r.Organization.ID, summary.BilledMicros, "restore", &actor)
	require.NoError(t, err)

	restored, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, summary.GrantMicros+summary.BilledMicros, restored.GrantMicros)
	assert.Equal(t, summary.GrantMicros, restored.RemainingMicros)
}

func Test__HostedRecordUsageAppliesOrgMarkupOverride(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	override := 0
	require.NoError(t, models.UpsertOrganizationLLMMarkup(db, r.Organization.ID, &override))

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

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
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
	modelBilledMicros := models.ApplyMarkupMicros(300*models.MicrosPerCent, models.DefaultMarkupBPS)

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

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, modelBilledMicros+computeCostMicros, summary.BilledMicros)
	assert.Equal(t, summary.GrantMicros-summary.BilledMicros, summary.RemainingMicros)
}

func Test__ComputeUsageAloneCanExhaustHostedCredit(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	runID := requireExecutionRunID(t, execution)

	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
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

	err = models.AssertHostedCreditAvailable(db, r.Organization.ID)
	require.ErrorIs(t, err, models.ErrHostedCreditEmpty)

	err = models.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	require.ErrorIs(t, err, models.ErrHostedCreditEmpty)
}

func Test__AssertHostedCreditAvailable(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	require.NoError(t, models.AssertHostedCreditAvailable(db, r.Organization.ID))

	_, err := models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		WelcomeGrantCents:   0,
		MarkupBPS:           models.DefaultMarkupBPS,
		WarningThresholdBPS: models.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)

	org, err := models.CreateOrganization(support.RandomName("empty-credit"), "")
	require.NoError(t, err)
	err = models.AssertHostedCreditAvailable(db, org.ID)
	require.ErrorIs(t, err, models.ErrHostedCreditEmpty)
}

func Test__AssertHostedRunAllowedAllowsConcurrentStarts(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()

	errs := make(chan error, 2)
	go func() {
		errs <- models.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	}()
	go func() {
		errs <- models.AssertHostedRunAllowed(db, r.Organization.ID, nil)
	}()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
}

func Test__PolarGrantIsIdempotentByOrderID(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()

	first, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
	require.NoError(t, err)
	second, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(2500), summary.GrantMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, models.CentsToMicros(2500), summary.PurchasedCreditMicros)
}

func Test__DescribeOrganizationLLMCreditSplitsSuperPlaneAndPurchasedGrants(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	_, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, models.CentsToMicros(10000), summary.PurchasedCreditMicros)
	assert.Equal(t, summary.SuperPlaneGrantMicros+summary.PurchasedCreditMicros, summary.GrantMicros)
	assert.Equal(t, summary.GrantMicros, summary.RemainingMicros)
}

func Test__PolarRefundIsIdempotentAndCapsAtGrant(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	refundID := uuid.NewString()

	_, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
	require.NoError(t, err)

	first, err := models.AddPolarLLMCreditRefund(db, r.Organization.ID, models.CentsToMicros(1000), orderID, refundID)
	require.NoError(t, err)
	second, err := models.AddPolarLLMCreditRefund(db, r.Organization.ID, models.CentsToMicros(1000), orderID, refundID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	reversed, err := models.PolarRefundMicrosForOrder(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(1000), reversed)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(1500), summary.GrantMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.SuperPlaneGrantMicros)
	assert.Equal(t, models.CentsToMicros(1500), summary.PurchasedCreditMicros)
}

func Test__ReversePolarOrderCreditDoesNotOverDebitConcurrently(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	_, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
	require.NoError(t, err)

	errs := make(chan error, 2)
	go func() {
		errs <- models.ReversePolarOrderCredit(db, r.Organization.ID, orderID, models.CentsToMicros(2500), orderID+":full")
	}()
	go func() {
		errs <- models.ReversePolarOrderCredit(db, r.Organization.ID, orderID, models.CentsToMicros(1000), orderID+":partial")
	}()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)

	reversed, err := models.PolarRefundMicrosForOrder(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(2500), reversed)
}

func Test__FactoryHostedBudgetZeroBlocksHostedStart(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.Conn()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	zero := int64(0)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &zero))

	err = models.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID)
	require.ErrorIs(t, err, models.ErrFactoryHostedBudgetEmpty)
}

func Test__FactoryHostedBudgetHardStopWhenSpent(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
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

	err = models.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID)
	require.ErrorIs(t, err, models.ErrFactoryHostedBudgetEmpty)
}

func Test__FactoryHostedBudgetHardStopWhenSpentOnCompute(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
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

	err = models.AssertHostedRunAllowed(db, r.Organization.ID, &factory.ID)
	require.ErrorIs(t, err, models.ErrFactoryHostedBudgetEmpty)
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
	_, err := models.UpdateInstallationLLMSettings(database.Conn(), models.InstallationLLMSettings{
		WelcomeGrantCents:   models.DefaultWelcomeGrantCents,
		MarkupBPS:           models.DefaultMarkupBPS,
		WarningThresholdBPS: models.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)
}

func Test__HostedLLMProviderAllowlist(t *testing.T) {
	_ = support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	saved, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	assert.True(t, saved.AllowsModel("claude-sonnet-4-6"))
	assert.False(t, saved.AllowsModel("gpt-5-mini"))

	disabled, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       false,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"openai/gpt-4.1"},
	})
	require.NoError(t, err)
	assert.True(t, disabled.OffersHostedModels())
	_, err = models.RequireEnabledHostedLLMProvider(db, models.UsageProviderOpenRouter)
	require.NoError(t, err)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{Provider: "bedrock"})
	require.Error(t, err)
}

func Test__HasOfferedHostedLLMProvider(t *testing.T) {
	_ = support.Setup(t)
	db := database.Conn()
	existing, err := models.ListHostedLLMProviders(db)
	require.NoError(t, err)
	require.NoError(t, db.Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		_ = database.Conn().Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error
		for _, provider := range existing {
			_, _ = models.UpsertHostedLLMProvider(database.Conn(), provider)
		}
	})

	offered, err := models.HasOfferedHostedLLMProvider(db)
	require.NoError(t, err)
	assert.False(t, offered)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderAnthropic,
		Enabled:  true,
		APIKey:   []byte("encrypted"),
	})
	require.NoError(t, err)
	offered, err = models.HasOfferedHostedLLMProvider(db)
	require.NoError(t, err)
	assert.False(t, offered)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	offered, err = models.HasOfferedHostedLLMProvider(db)
	require.NoError(t, err)
	assert.True(t, offered)
}

func Test__CentsToMicros(t *testing.T) {
	assert.Equal(t, int64(50_000_000), models.CentsToMicros(5000))
	assert.Equal(t, int64(0), models.CentsToMicros(0))
}

func Test__SignedMicrosToCents(t *testing.T) {
	assert.Equal(t, int64(2500), models.SignedMicrosToCents(models.CentsToMicros(2500)))
	assert.Equal(t, int64(-1000), models.SignedMicrosToCents(-models.CentsToMicros(1000)))
	assert.Equal(t, int64(0), models.SignedMicrosToCents(0))
}

func Test__ListOrganizationLLMCreditGrants(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	actor := r.Account.ID
	_, err := models.AddAdminLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(1500), "Support grant", &actor)
	require.NoError(t, err)

	orderID := uuid.NewString()
	_, err = models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
	require.NoError(t, err)
	_, err = models.AddPolarLLMCreditRefund(db, r.Organization.ID, models.CentsToMicros(500), orderID, orderID+":partial")
	require.NoError(t, err)

	grants, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(grants), 4)
	assert.Equal(t, models.LLMCreditGrantKindTopupRefund, grants[0].Kind)
	assert.Equal(t, models.LLMCreditGrantKindTopup, grants[1].Kind)
	assert.Equal(t, models.LLMCreditGrantKindAdmin, grants[2].Kind)
	assert.Equal(t, models.LLMCreditGrantKindWelcome, grants[3].Kind)

	names, err := models.CreditGrantActorNames(db, grants)
	require.NoError(t, err)
	assert.Equal(t, r.Account.Name, names[actor])
}

func Test__ExpireOpenIncludedGrantsLeavesTopupSpendable(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	_, err := models.AddTopupLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(2500),
		orderID,
	)
	require.NoError(t, err)

	periodEnd := time.Now().AddDate(0, 1, 0)
	includedKey := models.IncludedGrantKey("sub_expire", periodEnd)
	_, err = models.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(models.DefaultIncludedGrantCents),
		includedKey,
		periodEnd,
	)
	require.NoError(t, err)

	require.NoError(t, models.ExpireOpenIncludedGrants(db, r.Organization.ID))
	require.NoError(t, models.ExpireOpenIncludedGrants(db, r.Organization.ID))

	grant := findCanceledIncludedGrant(t, db, r.Organization.ID)
	require.NotNil(t, grant.ExpiresAt)
	assert.False(t, grant.ExpiresAt.After(time.Now()))

	_, err = models.FindLLMCreditGrantByPolarOrderID(db, includedKey)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(2500), summary.PurchasedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), summary.WelcomeRemainingMicros)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(2500), summary.RemainingMicros)
}

func Test__ExpireOpenIncludedGrantsAfterResubscribeSameKey(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	periodEnd := time.Now().AddDate(0, 1, 0)
	includedKey := models.IncludedGrantKey("sub_expire_again", periodEnd)

	_, err := models.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(models.DefaultIncludedGrantCents),
		includedKey,
		periodEnd,
	)
	require.NoError(t, err)
	require.NoError(t, models.ExpireOpenIncludedGrants(db, r.Organization.ID))

	_, err = models.AddIncludedLLMCreditGrant(
		db,
		r.Organization.ID,
		models.CentsToMicros(models.DefaultIncludedGrantCents),
		includedKey,
		periodEnd,
	)
	require.NoError(t, err)
	require.NoError(t, models.ExpireOpenIncludedGrants(db, r.Organization.ID))

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.IncludedRemainingMicros)

	_, err = models.FindLLMCreditGrantByPolarOrderID(db, includedKey)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func findCanceledIncludedGrant(t *testing.T, db *gorm.DB, orgID uuid.UUID) models.OrganizationLLMCreditGrant {
	t.Helper()
	grants, err := models.ListOrganizationLLMCreditGrants(db, orgID)
	require.NoError(t, err)
	now := time.Now()
	for _, grant := range grants {
		if grant.Kind != models.LLMCreditGrantKindIncluded {
			continue
		}
		if !grant.IsExpired(now) {
			continue
		}
		require.NotNil(t, grant.PolarOrderID)
		return grant
	}
	require.FailNow(t, "canceled included grant not found")
	return models.OrganizationLLMCreditGrant{}
}
