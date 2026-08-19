package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestSerializeWorkOrderCreator_UserBranch(t *testing.T) {
	userID := uuid.New()
	order := &models.FactoryWorkOrder{
		ID:          uuid.New(),
		CreatedByID: &userID,
		CreatedBy:   &models.User{Name: "Alice"},
	}

	creator := serializeWorkOrder(nil, order, nil, nil).GetCreatedBy()
	require.NotNil(t, creator)
	assert.Nil(t, creator.GetAutomation())
	require.NotNil(t, creator.GetUser())
	assert.Equal(t, userID.String(), creator.GetUser().GetId())
	assert.Equal(t, "Alice", creator.GetUser().GetName())
}

func TestSerializeWorkOrderCreator_AutomationBranchWinsOverUser(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()
	order := &models.FactoryWorkOrder{
		ID:          uuid.New(),
		CreatedByID: &userID,
		CreatedBy:   &models.User{Name: "Alice"},
	}
	automation := &factory.AutomationRef{
		NodeID:   "node-1",
		NodeName: "Release gate",
		AppID:    appID,
		AppName:  "Release automation",
	}

	creator := serializeWorkOrder(nil, order, nil, automation).GetCreatedBy()
	require.NotNil(t, creator)
	assert.Nil(t, creator.GetUser())
	require.NotNil(t, creator.GetAutomation())
	assert.Equal(t, "node-1", creator.GetAutomation().GetNodeId())
	assert.Equal(t, "Release gate", creator.GetAutomation().GetNodeName())
	assert.Equal(t, appID.String(), creator.GetAutomation().GetAppId())
}

func TestSerializeWorkOrderCreator_NoneReturnsNil(t *testing.T) {
	order := &models.FactoryWorkOrder{ID: uuid.New()}
	assert.Nil(t, serializeWorkOrder(nil, order, nil, nil).GetCreatedBy())
}

func TestSerializeExecutionSteps_UsesCanvasNames(t *testing.T) {
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp},
		{Type: models.FactoryLineStepTypeRunApp},
		{Type: models.FactoryLineStepTypeRunApp},
	}
	executions := []models.FactoryWorkOrderExecutionRecord{
		{FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{StepIndex: 0}, CanvasName: "plan-app"},
		{FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{StepIndex: 1}, CanvasName: "implement-app"},
		{FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{StepIndex: 2}, CanvasName: "verify-app"},
	}

	out := serializeExecutionSteps(steps, executions)
	require.Len(t, out, 3)
	assert.Equal(t, "plan-app", out[0].GetName())
	assert.EqualValues(t, 0, out[0].GetStepIndex())
	assert.Equal(t, "implement-app", out[1].GetName())
	assert.EqualValues(t, 1, out[1].GetStepIndex())
	assert.Equal(t, "verify-app", out[2].GetName())
	assert.EqualValues(t, 2, out[2].GetStepIndex())
}

func TestSerializeExecutionSteps_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, serializeExecutionSteps(nil, nil))
	assert.Nil(t, serializeExecutionSteps([]models.FactoryLineStep{}, nil))
}

func TestSerializeWorkOrder_LineDispatchesReplaceFlatExecutions(t *testing.T) {
	lineID := uuid.New()
	dispatchID := uuid.New()
	runID := uuid.New()
	canvasID := uuid.New()

	dispatches := []models.FactoryWorkOrderLineDispatchRecord{
		{
			FactoryWorkOrderLineDispatch: models.FactoryWorkOrderLineDispatch{
				ID:       dispatchID,
				LineID:   lineID,
				LineName: "ship",
				Steps: datatypes.NewJSONSlice([]models.FactoryLineStep{
					{Type: models.FactoryLineStepTypeRunApp},
					{Type: models.FactoryLineStepTypeRunApp},
				}),
				State: models.FactoryWorkOrderLineDispatchStateActive,
			},
			Executions: []models.FactoryWorkOrderExecutionRecord{
				{
					FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
						ID:          uuid.New(),
						StepIndex:   0,
						StepName:    "step-one",
						RunID:       runID,
						Status:      models.FactoryWorkOrderExecutionStatusFinished,
						Result:      models.CanvasRunResultPassed,
						TotalTokens: 10,
						CostCents:   5,
					},
					CanvasID:   canvasID,
					CanvasName: "step-one-app",
				},
			},
		},
	}

	order := &models.FactoryWorkOrder{ID: uuid.New()}
	serialized := serializeWorkOrder(nil, order, dispatches, nil)

	require.Len(t, serialized.LineDispatches, 1)
	dispatch := serialized.LineDispatches[0]
	assert.Equal(t, dispatchID.String(), dispatch.Id)
	assert.Equal(t, lineID.String(), dispatch.Line.Id)
	assert.Equal(t, "ship", dispatch.Line.Name)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, dispatch.State)
	require.Len(t, dispatch.Steps, 2)
	assert.Equal(t, "step-one-app", dispatch.Steps[0].Name)
	assert.Equal(t, "", dispatch.Steps[1].Name)

	require.Len(t, dispatch.StepExecutions, 1)
	execution := dispatch.StepExecutions[0]
	assert.Equal(t, "step-one", execution.Step)
	assert.Equal(t, pb.WorkOrderExecution_STATE_FINISHED, execution.State)
	assert.Equal(t, pb.WorkOrderExecution_RESULT_PASSED, execution.Result)

	// Aggregate usage sums across every dispatch's step executions.
	assert.EqualValues(t, 10, serialized.TotalTokens)
	assert.EqualValues(t, 5, serialized.TotalCostCents)
}
