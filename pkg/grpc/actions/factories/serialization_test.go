package factories

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func mustSerializeWorkOrder(
	t *testing.T,
	f *models.Factory,
	order *models.FactoryWorkOrder,
	dispatches []models.FactoryWorkOrderLineDispatchRecord,
	createdByAutomation *factory.AutomationRef,
) *pb.WorkOrder {
	t.Helper()
	serialized, err := serializeWorkOrder(f, order, dispatches, createdByAutomation)
	require.NoError(t, err)
	return serialized
}

func TestSerializeWorkOrderCreator_UserBranch(t *testing.T) {
	userID := uuid.New()
	order := &models.FactoryWorkOrder{
		ID:          uuid.New(),
		CreatedByID: &userID,
		CreatedBy:   &models.User{Name: "Alice"},
	}

	creator := mustSerializeWorkOrder(t, nil, order, nil, nil).GetCreatedBy()
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

	creator := mustSerializeWorkOrder(t, nil, order, nil, automation).GetCreatedBy()
	require.NotNil(t, creator)
	assert.Nil(t, creator.GetUser())
	require.NotNil(t, creator.GetAutomation())
	assert.Equal(t, "node-1", creator.GetAutomation().GetNodeId())
	assert.Equal(t, "Release gate", creator.GetAutomation().GetNodeName())
	assert.Equal(t, appID.String(), creator.GetAutomation().GetAppId())
}

func TestSerializeWorkOrderCreator_NoneReturnsNil(t *testing.T) {
	order := &models.FactoryWorkOrder{ID: uuid.New()}
	assert.Nil(t, mustSerializeWorkOrder(t, nil, order, nil, nil).GetCreatedBy())
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

func TestSerializeExecutionSteps_FallsBackToStepNameWhenCanvasGone(t *testing.T) {
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp},
		{Type: models.FactoryLineStepTypeRunApp},
	}
	executions := []models.FactoryWorkOrderExecutionRecord{
		{
			FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
				StepIndex: 0,
				StepName:  "implement",
			},
		},
		{
			FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
				StepIndex: 1,
				StepName:  "verify",
			},
			CanvasName: "verify-app",
		},
	}

	out := serializeExecutionSteps(steps, executions)
	require.Len(t, out, 2)
	assert.Equal(t, "implement", out[0].GetName())
	assert.Equal(t, "verify-app", out[1].GetName())
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
						RunID:       &runID,
						Status:      models.FactoryWorkOrderExecutionStatusFinished,
						Result:      models.CanvasRunResultPassed,
						TotalTokens: 10,
						CostCents:   5,
					},
					CanvasID:   &canvasID,
					CanvasName: "step-one-app",
				},
			},
		},
	}

	order := &models.FactoryWorkOrder{ID: uuid.New()}
	serialized := mustSerializeWorkOrder(t, nil, order, dispatches, nil)

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

func TestSerializeWorkOrder_StatusNotes(t *testing.T) {
	appID := uuid.New()
	runID := uuid.New()
	note, err := json.Marshal([]models.FactoryWorkOrderStatusNote{
		{
			Key:      "pr-closure",
			Kind:     models.FactoryWorkOrderStatusNoteKindInfo,
			Headline: "Review the pull request",
			Body:     "Merging PR #42 completes this work order.",
			CtaLabel: "Review PR #42",
			CtaURL:   "https://github.com/acme/app/pull/42",
			Automation: &factory.AutomationRef{
				AppID:   appID,
				AppName: "PR Closure",
			},
			Run:       &factory.RunRef{ID: runID},
			UpdatedAt: time.Now(),
		},
	})
	require.NoError(t, err)

	order := &models.FactoryWorkOrder{ID: uuid.New(), StatusNote: note}
	serialized := mustSerializeWorkOrder(t, nil, order, nil, nil)

	statusNotes := serialized.GetStatusNotes()
	require.Len(t, statusNotes, 1)
	statusNote := statusNotes[0]
	assert.Equal(t, "pr-closure", statusNote.GetKey())
	assert.Equal(t, "info", statusNote.GetKind())
	assert.Equal(t, "Review the pull request", statusNote.GetHeadline())
	assert.Equal(t, "Review PR #42", statusNote.GetCtaLabel())
	assert.Equal(t, "https://github.com/acme/app/pull/42", statusNote.GetCtaUrl())
	require.NotNil(t, statusNote.GetAutomation())
	assert.Equal(t, appID.String(), statusNote.GetAutomation().GetAppId())
	assert.Equal(t, runID.String(), statusNote.GetRunId())
	assert.NotNil(t, statusNote.GetUpdatedAt())

	// No notes stored serializes as absent, not as an empty list.
	bare := &models.FactoryWorkOrder{ID: uuid.New()}
	assert.Empty(t, mustSerializeWorkOrder(t, nil, bare, nil, nil).GetStatusNotes())
}

func TestSerializeWorkOrderExecution_OmitsRunWhenRunIDNil(t *testing.T) {
	now := time.Now()
	out := serializeWorkOrderExecution(models.FactoryWorkOrderExecutionRecord{
		FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
			ID:        uuid.New(),
			LineID:    uuid.New(),
			StepName:  "implement",
			StepIndex: 1,
			Status:    models.FactoryWorkOrderExecutionStatusFinished,
			Result:    models.CanvasRunResultPassed,
			CreatedAt: now,
			UpdatedAt: now,
		},
	})

	assert.Nil(t, out.GetRun())
	assert.Equal(t, pb.WorkOrderExecution_STATE_FINISHED, out.GetState())
	assert.Equal(t, pb.WorkOrderExecution_RESULT_PASSED, out.GetResult())
	assert.Equal(t, "implement", out.GetStep())
}

func TestSerializeWorkOrderExecution_IncludesRunWhenRunIDSet(t *testing.T) {
	runID := uuid.New()
	canvasID := uuid.New()
	now := time.Now()
	out := serializeWorkOrderExecution(models.FactoryWorkOrderExecutionRecord{
		FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
			ID:        uuid.New(),
			LineID:    uuid.New(),
			StepName:  "implement",
			RunID:     &runID,
			Status:    models.FactoryWorkOrderExecutionStatusFinished,
			Result:    models.CanvasRunResultPassed,
			CreatedAt: now,
			UpdatedAt: now,
		},
		CanvasID:   &canvasID,
		CanvasName: "Implement app",
	})

	require.NotNil(t, out.GetRun())
	assert.Equal(t, runID.String(), out.GetRun().GetId())
	assert.Equal(t, canvasID.String(), out.GetRun().GetAppId())
	assert.Equal(t, "Implement app", out.GetRun().GetAppName())
}
