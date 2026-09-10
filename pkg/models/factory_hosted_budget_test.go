package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SumFactoryHostedBilledMicrosIncludesComputeAndModel(t *testing.T) {
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
		MachineType:     "e1-tiny-arm64",
		FleetID:         "e1-tiny-arm64",
		DurationSeconds: 3600,
	}))
	var computeCostMicros int64
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Select("COALESCE(SUM(cost_micros), 0)").
		Where("canvas_run_id = ? AND usage_kind = ?", runID, models.UsageKindCompute).
		Scan(&computeCostMicros).Error)
	require.Greater(t, computeCostMicros, int64(0))

	billed, err := models.SumFactoryHostedBilledMicros(db, r.Organization.ID, execution.FactoryID)
	require.NoError(t, err)
	assert.Equal(t, modelBilledMicros+computeCostMicros, billed)
}

func Test__DescribeFactoryHostedBudgetAccountsForComputeSpend(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	factory, err := models.FindFactory(db, r.Organization.ID, execution.FactoryID)
	require.NoError(t, err)

	budgetCents := int64(100)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &budgetCents))

	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 3600,
	}))

	summary, err := models.DescribeFactoryHostedBudget(db, factory)
	require.NoError(t, err)
	require.True(t, summary.Capped)
	require.Greater(t, summary.BilledMicros, models.CentsToMicros(budgetCents))
	assert.Equal(t, int64(0), summary.RemainingMicros)
}
