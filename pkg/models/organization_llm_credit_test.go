package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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

	require.NoError(t, models.GrantWelcomeCredit(db, r.Organization.ID))
	again, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, summary.GrantMicros, again.GrantMicros)
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

	org, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.GrantMicros)
}

func Test__AdminGrantRestoresHostedCredit(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	execution := dispatchWorkOrderExecution(t, r)
	require.NoError(t, models.RecordUsage(db, models.LLMUsageEventInput{
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

	require.NoError(t, models.RecordUsage(db, models.LLMUsageEventInput{
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

	var event models.LLMUsageEvent
	require.NoError(t, db.Where("work_order_execution_id = ?", execution.ID).First(&event).Error)
	assert.Equal(t, models.UsageFundingSourceHosted, event.FundingSource)
	assert.Equal(t, int64(3_000_000), event.ProviderCostMicros)
	assert.Equal(t, int64(3_000_000), event.CostMicros)
}

func Test__BYOKRecordUsageIsNotMarkedUp(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	require.NoError(t, models.RecordUsage(db, models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))

	var event models.LLMUsageEvent
	require.NoError(t, db.Where("work_order_execution_id = ?", execution.ID).First(&event).Error)
	assert.Equal(t, models.UsageFundingSourceBYOK, event.FundingSource)
	assert.Equal(t, event.ProviderCostMicros, event.CostMicros)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), summary.BilledMicros)
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
