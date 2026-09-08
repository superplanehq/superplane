package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestValidateSpendingReportWindow(t *testing.T) {
	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()
	require.NoError(t, models.ValidateSpendingReportWindow(start, end))

	err := models.ValidateSpendingReportWindow(end, start)
	require.Error(t, err)

	err = models.ValidateSpendingReportWindow(start, start.AddDate(0, 0, 400))
	require.Error(t, err)
}

func TestSummarizeSpendingKPITotalsAndExplorer(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))

	require.NotNil(t, execution.RunID)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		FundingSource:   models.UsageFundingSourceHosted,
		InputTokens:     1000,
		TotalTokens:     1000,
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-tiny-amd64",
		FleetID:         "e1-tiny-amd64",
		DurationSeconds: 30,
		IdempotencyKey:  "runner:compute:spending-report",
	}))

	since := time.Now().AddDate(0, 0, -7)
	until := time.Now().Add(time.Hour)
	filter := models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          since,
		Until:          until,
	}

	kpi, err := models.SummarizeSpendingKPITotals(db, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), kpi.TotalTokens)
	assert.Equal(t, int64(30), kpi.DurationSeconds)
	assert.Positive(t, kpi.CostMicros)
	assert.Positive(t, kpi.HostedCostMicros)

	modelFilter := filter
	modelFilter.UsageKind = models.UsageKindModel
	modelFilter.FactoryID = &factory.ID
	explorer, err := models.SummarizeSpendingExplorer(db, modelFilter, models.SpendingGroupByWorkspace, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), explorer.Totals.TotalTokens)
	require.NotEmpty(t, explorer.Breakdown)
	assert.Equal(t, factory.ID.String(), explorer.Breakdown[0].ID)
	require.NotEmpty(t, explorer.Series)

	catalogs, err := models.ListSpendingFilterCatalogs(db, filter)
	require.NoError(t, err)
	require.Len(t, catalogs.Workspaces, 1)
	assert.Equal(t, factory.Name, catalogs.Workspaces[0].Label)
	require.Len(t, catalogs.Users, 1)
	assert.Equal(t, r.UserModel.Name, catalogs.Users[0].Label)
}

func TestSummarizeSpendingKPITotalsIgnoresFactoryFilter(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	recordFactoryUsage := func(factory *models.Factory, tokens int64, idempotencySuffix string) {
		order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
		require.NoError(t, err)

		line, err := factory.CreateLine(db, "ship", nil)
		require.NoError(t, err)

		app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
		require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
		}, nil))

		var execution *models.FactoryWorkOrderExecution
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			_, result, dispatchErr := line.Dispatch(tx, order)
			if dispatchErr != nil {
				return dispatchErr
			}
			execution = result.Execution
			return nil
		}))

		require.NotNil(t, execution.RunID)
		require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
			OrganizationID:  r.Organization.ID,
			CanvasRunID:     *execution.RunID,
			NodeExecutionID: uuid.New(),
			NodeID:          "prompt",
			Provider:        models.UsageProviderOpenAI,
			Model:           "gpt-4o",
			InputTokens:     tokens,
			TotalTokens:     tokens,
			IdempotencyKey:  "spending-kpi-model:" + idempotencySuffix,
		}))
	}

	factoryA, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory-a"), "", "")
	require.NoError(t, err)
	factoryB, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory-b"), "", "")
	require.NoError(t, err)

	recordFactoryUsage(factoryA, 1000, factoryA.ID.String())
	recordFactoryUsage(factoryB, 4000, factoryB.ID.String())

	since := time.Now().AddDate(0, 0, -7)
	until := time.Now().Add(time.Hour)
	windowFilter := models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          since,
		Until:          until,
	}

	kpi, err := models.SummarizeSpendingKPITotals(db, windowFilter)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), kpi.TotalTokens)

	filtered := windowFilter
	filtered.FactoryID = &factoryA.ID
	filtered.UsageKind = models.UsageKindModel
	explorer, err := models.SummarizeSpendingExplorer(db, filtered, models.SpendingGroupByWorkspace, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), explorer.Totals.TotalTokens)
}

func TestSummarizeSpendingExplorerFiltersByTaskOwner(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))

	require.NotNil(t, execution.RunID)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     500,
		TotalTokens:     500,
	}))

	since := time.Now().AddDate(0, 0, -1)
	until := time.Now().Add(time.Hour)
	filter := models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          since,
		Until:          until,
		UsageKind:      models.UsageKindModel,
		TaskOwnerID:    &r.User,
	}

	explorer, err := models.SummarizeSpendingExplorer(db, filter, models.SpendingGroupByUser, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	require.Len(t, explorer.Breakdown, 1)
	assert.Equal(t, r.User.String(), explorer.Breakdown[0].ID)
}

func TestSpendingModelDisplayName(t *testing.T) {
	allowlist := []string{"claude-haiku-4-5", "claude-opus-4-6", "claude-sonnet-4-6"}

	assert.Equal(t, "claude-sonnet-4-6", models.SpendingModelDisplayName("sonnet", allowlist))
	assert.Equal(t, "claude-opus-4-6", models.SpendingModelDisplayName("opus", allowlist))
	assert.Equal(t, "claude-sonnet-4-6", models.SpendingModelDisplayName("claude-sonnet-4-6", allowlist))
	assert.Equal(t, "claude-sonnet-4-6", models.SpendingModelDisplayName("sonnet", []string{"anthropic/claude-sonnet-4-6"}))
	assert.Equal(t, "claude-opus-4-6", models.SpendingModelDisplayName("opus", []string{"anthropic/claude-opus-4-6", "anthropic/claude-sonnet-4-6"}))
	assert.Equal(t, "gpt-4o", models.SpendingModelDisplayName("gpt-4o", nil))
	assert.Equal(t, "sonnet", models.SpendingModelDisplayName("sonnet", nil))
	assert.Equal(t, "sonnet", models.SpendingModelDisplayName("sonnet", []string{"sonnet"}))
}

func TestSpendingBreakdownLabelUsesFundingSourceNames(t *testing.T) {
	assert.Equal(t, "SuperPlane-hosted", models.SpendingBreakdownLabel("hosted", models.SpendingCatalogs{}, models.SpendingGroupByFundingSource))
	assert.Equal(t, "Your keys", models.SpendingBreakdownLabel("byok", models.SpendingCatalogs{}, models.SpendingGroupByFundingSource))
}

func TestParseUsageFundingSource(t *testing.T) {
	hosted, err := models.ParseUsageFundingSource("Hosted")
	require.NoError(t, err)
	assert.Equal(t, models.UsageFundingSourceHosted, hosted)

	byok, err := models.ParseUsageFundingSource("byok")
	require.NoError(t, err)
	assert.Equal(t, models.UsageFundingSourceBYOK, byok)

	_, err = models.ParseUsageFundingSource("")
	require.Error(t, err)
	_, err = models.ParseUsageFundingSource("wallet")
	require.Error(t, err)
}

func TestSummarizeSpendingExplorerSplitsHostedAndBYOK(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, runID := dispatchSpendingFactoryRun(t, r, db)
	byokMicros := int64(2_500_000)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "hosted-prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		FundingSource:   models.UsageFundingSourceHosted,
		InputTokens:     1000,
		TotalTokens:     1000,
		IdempotencyKey:  "spending-split-hosted:" + factory.ID.String(),
	}))
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "byok-prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		FundingSource:   models.UsageFundingSourceBYOK,
		InputTokens:     400,
		TotalTokens:     400,
		CostMicros:      &byokMicros,
		IdempotencyKey:  "spending-split-byok:" + factory.ID.String(),
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-tiny-amd64",
		FleetID:         "e1-tiny-amd64",
		DurationSeconds: 30,
		IdempotencyKey:  "spending-split-compute:" + factory.ID.String(),
	}))

	since := time.Now().AddDate(0, 0, -7)
	until := time.Now().Add(time.Hour)
	window := models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          since,
		Until:          until,
	}

	kpi, err := models.SummarizeSpendingKPITotals(db, window)
	require.NoError(t, err)
	assert.Equal(t, int64(1400), kpi.TotalTokens)
	assert.Equal(t, int64(30), kpi.DurationSeconds)
	assert.Equal(t, byokMicros, kpi.BYOKCostMicros)
	assert.Equal(t, kpi.CostMicros-kpi.BYOKCostMicros, kpi.HostedCostMicros)
	assert.Positive(t, kpi.HostedCostMicros)

	modelFilter := window
	modelFilter.UsageKind = models.UsageKindModel
	grouped, err := models.SummarizeSpendingExplorer(db, modelFilter, models.SpendingGroupByFundingSource, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	assert.Equal(t, int64(1400), grouped.Totals.TotalTokens)
	assert.Equal(t, byokMicros, grouped.Totals.BYOKCostMicros)
	assert.Equal(t, grouped.Totals.CostMicros-grouped.Totals.BYOKCostMicros, grouped.Totals.HostedCostMicros)
	assert.Positive(t, grouped.Totals.HostedCostMicros)
	assert.Greater(t, kpi.HostedCostMicros, grouped.Totals.HostedCostMicros)
	require.Len(t, grouped.Breakdown, 2)
	assert.ElementsMatch(t, []string{models.UsageFundingSourceHosted, models.UsageFundingSourceBYOK}, []string{
		grouped.Breakdown[0].ID,
		grouped.Breakdown[1].ID,
	})
	assert.Equal(t, "SuperPlane-hosted", models.SpendingBreakdownLabel(models.UsageFundingSourceHosted, models.SpendingCatalogs{}, models.SpendingGroupByFundingSource))
	assert.Equal(t, "Your keys", models.SpendingBreakdownLabel(models.UsageFundingSourceBYOK, models.SpendingCatalogs{}, models.SpendingGroupByFundingSource))

	byokFilter := modelFilter
	byokFilter.FundingSource = models.UsageFundingSourceBYOK
	byokExplorer, err := models.SummarizeSpendingExplorer(db, byokFilter, models.SpendingGroupByFundingSource, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	assert.Equal(t, int64(400), byokExplorer.Totals.TotalTokens)
	assert.Equal(t, byokMicros, byokExplorer.Totals.BYOKCostMicros)
	assert.Equal(t, int64(0), byokExplorer.Totals.HostedCostMicros)
	require.Len(t, byokExplorer.Breakdown, 1)
	assert.Equal(t, models.UsageFundingSourceBYOK, byokExplorer.Breakdown[0].ID)
}

func TestSummarizeSpendingExplorerTreatsBlankFundingSourceAsBYOK(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, runID := dispatchSpendingFactoryRun(t, r, db)
	byokMicros := int64(2_500_000)

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "blank-source",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		FundingSource:   models.UsageFundingSourceBYOK,
		InputTokens:     400,
		TotalTokens:     400,
		CostMicros:      &byokMicros,
		IdempotencyKey:  "spending-blank-source:" + factory.ID.String(),
	}))
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("idempotency_key = ?", "spending-blank-source:"+factory.ID.String()).
		Updates(map[string]any{"funding_source": ""}).Error)

	since := time.Now().AddDate(0, 0, -7)
	until := time.Now().Add(time.Hour)
	filter := models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          since,
		Until:          until,
		UsageKind:      models.UsageKindModel,
	}

	grouped, err := models.SummarizeSpendingExplorer(db, filter, models.SpendingGroupByFundingSource, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	require.Len(t, grouped.Breakdown, 1)
	assert.Equal(t, models.UsageFundingSourceBYOK, grouped.Breakdown[0].ID)
	assert.Equal(t, byokMicros, grouped.Totals.BYOKCostMicros)

	filter.FundingSource = models.UsageFundingSourceBYOK
	byokExplorer, err := models.SummarizeSpendingExplorer(db, filter, models.SpendingGroupByFundingSource, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	assert.Equal(t, int64(400), byokExplorer.Totals.TotalTokens)
	require.Len(t, byokExplorer.Breakdown, 1)

	filter.FundingSource = models.UsageFundingSourceHosted
	hostedExplorer, err := models.SummarizeSpendingExplorer(db, filter, models.SpendingGroupByFundingSource, models.SpendingTimeGrainDay)
	require.NoError(t, err)
	assert.Equal(t, int64(0), hostedExplorer.Totals.TotalTokens)
	assert.Empty(t, hostedExplorer.Breakdown)
}

func dispatchSpendingFactoryRun(t *testing.T, r *support.ResourceRegistry, db *gorm.DB) (*models.Factory, uuid.UUID) {
	t.Helper()

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))

	require.NotNil(t, execution)
	require.NotNil(t, execution.RunID)
	return factory, *execution.RunID
}

func TestListSpendingFilterCatalogsUsesVersionedModelLabels(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		_ = database.DB(t.Context()).Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))

	require.NotNil(t, execution.RunID)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "sonnet",
		FundingSource:   models.UsageFundingSourceHosted,
		InputTokens:     100,
		TotalTokens:     100,
		IdempotencyKey:  "spending-model-alias:" + factory.ID.String(),
	}))

	catalogs, err := models.ListSpendingFilterCatalogs(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          time.Now().AddDate(0, 0, -1),
		Until:          time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.Len(t, catalogs.Models, 1)
	assert.Equal(t, "anthropic/sonnet", catalogs.Models[0].ID)
	assert.Equal(t, "claude-sonnet-4-6", catalogs.Models[0].Label)
}

func TestListSpendingFilterCatalogsResolvesAliasFromAnyHostedAllowlist(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Where("provider IN ?", []string{models.UsageProviderAnthropic, models.UsageProviderOpenRouter}).Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		_ = database.DB(t.Context()).Where("provider IN ?", []string{models.UsageProviderAnthropic, models.UsageProviderOpenRouter}).Delete(&models.HostedLLMProvider{})
	})

	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"anthropic/claude-sonnet-4-6", "anthropic/claude-opus-4-6"},
	})
	require.NoError(t, err)

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))

	require.NotNil(t, execution.RunID)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "sonnet",
		FundingSource:   models.UsageFundingSourceHosted,
		InputTokens:     100,
		TotalTokens:     100,
		IdempotencyKey:  "spending-model-openrouter-alias:" + factory.ID.String(),
	}))

	catalogs, err := models.ListSpendingFilterCatalogs(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		Since:          time.Now().AddDate(0, 0, -1),
		Until:          time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.Len(t, catalogs.Models, 1)
	assert.Equal(t, "anthropic/sonnet", catalogs.Models[0].ID)
	assert.Equal(t, "claude-sonnet-4-6", catalogs.Models[0].Label)
}
