package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__DescribeWorkOrder_AcceptsNumberAndKey(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Numbers", "", "NUM")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "one", "", &r.User, nil, nil)
	require.NoError(t, err)

	t.Run("describes by task number", func(t *testing.T) {
		resp, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
			FactoryId: "num",
			OrderId:   "1",
		})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), resp.Order.GetId())
		assert.Equal(t, int64(1), resp.Order.GetNumber())
	})

	t.Run("describes by task key", func(t *testing.T) {
		resp, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   "num-1",
		})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), resp.Order.GetId())
	})
}

func Test__DescribeWorkOrder_IncludesUsageBreakdown(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Spend", "", "SPD")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "one", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "build", "start")
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

	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 90,
		IdempotencyKey:  "runner:compute:" + uuid.New().String(),
	}))

	resp, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Order)
	assert.Positive(t, resp.Order.GetTotalCostCents())
	require.Len(t, resp.Order.GetUsageByModel(), 1)
	assert.Equal(t, "anthropic", resp.Order.GetUsageByModel()[0].GetProvider())
	assert.Equal(t, "claude-sonnet-4-6", resp.Order.GetUsageByModel()[0].GetModel())
	assert.EqualValues(t, 1_000_000, resp.Order.GetUsageByModel()[0].GetTotalTokens())
	assert.Positive(t, resp.Order.GetUsageByModel()[0].GetCostCents())
	require.Len(t, resp.Order.GetUsageByMachineType(), 1)
	assert.Equal(t, "e1-large-amd64", resp.Order.GetUsageByMachineType()[0].GetMachineType())
	assert.EqualValues(t, 90, resp.Order.GetUsageByMachineType()[0].GetDurationSeconds())
	require.NotEmpty(t, resp.Order.GetLineDispatches())
	require.NotEmpty(t, resp.Order.GetLineDispatches()[0].GetStepExecutions())
	assert.Equal(t, []string{"anthropic/claude-sonnet-4-6"}, resp.Order.GetLineDispatches()[0].GetStepExecutions()[0].GetModels())
}

func Test__DescribeWorkOrder_IncludesPlanningSessionSummary(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	session := openAnalysisSession(t, r, database.DB(ctx))
	require.NotNil(t, session.DraftWorkOrderID)

	resp, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: session.FactoryID.String(),
		OrderId:   session.DraftWorkOrderID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Order.GetPlanningSession())
	assert.Equal(t, session.ID.String(), resp.Order.GetPlanningSession().GetId())
	assert.Equal(t, session.State, resp.Order.GetPlanningSession().GetState())
	assert.Empty(t, resp.Order.GetPlanningSession().GetSurvey().GetQuestions())
}
