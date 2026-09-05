package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test__DescribeOrganizationSpendingReport(t *testing.T) {
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
		InputTokens:     2000,
		TotalTokens:     2000,
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-tiny-amd64",
		FleetID:         "e1-tiny-amd64",
		DurationSeconds: 60,
		IdempotencyKey:  "runner:compute:spending-report-handler",
	}))

	end := time.Now()
	start := end.AddDate(0, 0, -7)

	resp, err := DescribeOrganizationSpendingReport(
		context.Background(),
		r.Organization.ID.String(),
		&pb.DescribeOrganizationSpendingReportRequest{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
			GroupBy:   models.SpendingGroupByWorkspace,
			TimeGrain: models.SpendingTimeGrainDay,
			UsageKind: models.UsageKindModel,
			FactoryId: factory.ID.String(),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, resp.KpiTotals)
	assert.Equal(t, int64(2000), resp.KpiTotals.TotalTokens)
	assert.Equal(t, int64(60), resp.KpiTotals.DurationSeconds)
	assert.Positive(t, resp.KpiTotals.CostCents)
	require.NotNil(t, resp.ExplorerTotals)
	assert.Equal(t, int64(2000), resp.ExplorerTotals.TotalTokens)
	require.NotEmpty(t, resp.Breakdown)
	assert.Equal(t, factory.ID.String(), resp.Breakdown[0].Id)
	assert.Equal(t, factory.Name, resp.Breakdown[0].Label)
	require.NotEmpty(t, resp.Series)
	require.NotNil(t, resp.Credit)
	assert.Positive(t, resp.Credit.RemainingCreditCents)
	require.NotNil(t, resp.Catalogs)
	assert.NotEmpty(t, resp.Catalogs.Workspaces)
}

func Test__DescribeOrganizationSpendingReport__KPITotalsIgnoreExplorerFilters(t *testing.T) {
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
			IdempotencyKey:  "spending-kpi:" + idempotencySuffix,
		}))
	}

	factoryA, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory-a"), "", "")
	require.NoError(t, err)
	factoryB, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory-b"), "", "")
	require.NoError(t, err)

	recordFactoryUsage(factoryA, 2000, factoryA.ID.String())
	recordFactoryUsage(factoryB, 3000, factoryB.ID.String())

	end := time.Now()
	start := end.AddDate(0, 0, -7)

	resp, err := DescribeOrganizationSpendingReport(
		context.Background(),
		r.Organization.ID.String(),
		&pb.DescribeOrganizationSpendingReportRequest{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
			GroupBy:   models.SpendingGroupByWorkspace,
			TimeGrain: models.SpendingTimeGrainDay,
			UsageKind: models.UsageKindModel,
			FactoryId: factoryA.ID.String(),
		},
	)
	require.NoError(t, err)
	require.NotNil(t, resp.KpiTotals)
	assert.Equal(t, int64(5000), resp.KpiTotals.TotalTokens)
	require.NotNil(t, resp.ExplorerTotals)
	assert.Equal(t, int64(2000), resp.ExplorerTotals.TotalTokens)
	require.NotEmpty(t, resp.Breakdown)
	assert.Equal(t, factoryA.ID.String(), resp.Breakdown[0].Id)
}

func Test__DescribeOrganizationSpendingReport__ModelBreakdownUsesVersionedLabels(t *testing.T) {
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
		InputTokens:     200,
		TotalTokens:     200,
		IdempotencyKey:  "spending-report-model-alias:" + factory.ID.String(),
	}))

	end := time.Now()
	start := end.AddDate(0, 0, -1)
	resp, err := DescribeOrganizationSpendingReport(
		context.Background(),
		r.Organization.ID.String(),
		&pb.DescribeOrganizationSpendingReportRequest{
			StartTime: timestamppb.New(start),
			EndTime:   timestamppb.New(end),
			GroupBy:   models.SpendingGroupByModel,
			TimeGrain: models.SpendingTimeGrainDay,
			UsageKind: models.UsageKindModel,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Breakdown)
	assert.Equal(t, "anthropic/sonnet", resp.Breakdown[0].Id)
	assert.Equal(t, "claude-sonnet-4-6", resp.Breakdown[0].Label)
	require.NotEmpty(t, resp.SeriesKeys)
	assert.Equal(t, "claude-sonnet-4-6", resp.SeriesKeys[0].Label)
	require.NotNil(t, resp.Catalogs)
	require.NotEmpty(t, resp.Catalogs.Models)
	assert.Equal(t, "claude-sonnet-4-6", resp.Catalogs.Models[0].Label)
}
