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
	serialized, err := serializeWorkOrder(f, order, dispatches, createdByAutomation, workOrderUsageView{})
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

func TestSerializeWorkOrder_SourceRunID(t *testing.T) {
	runID := uuid.New()
	order := &models.FactoryWorkOrder{ID: uuid.New(), SourceRunID: &runID}

	assert.Equal(t, runID.String(), mustSerializeWorkOrder(t, nil, order, nil, nil).GetSourceRunId())
	assert.Empty(t, mustSerializeWorkOrder(t, nil, &models.FactoryWorkOrder{ID: uuid.New()}, nil, nil).GetSourceRunId())
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
	serialized, err := serializeWorkOrder(nil, order, dispatches, nil, workOrderUsageView{
		Totals: models.UsageTotals{
			TotalTokens: 10,
			CostMicros:  50_000,
		},
	})
	require.NoError(t, err)

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

	// Work-order totals come from the ledger, not from step-cache sums.
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
		{
			Key:                 "queue-slot",
			Kind:                models.FactoryWorkOrderStatusNoteKindInfo,
			Headline:            "Waiting for a slot",
			ShowOnlyWhenWaiting: true,
			UpdatedAt:           time.Now(),
		},
	})
	require.NoError(t, err)

	order := &models.FactoryWorkOrder{ID: uuid.New(), StatusNote: note}
	serialized := mustSerializeWorkOrder(t, nil, order, nil, nil)

	statusNotes := serialized.GetStatusNotes()
	require.Len(t, statusNotes, 2)
	statusNote := statusNotes[0]
	assert.Equal(t, "pr-closure", statusNote.GetKey())
	assert.Equal(t, "info", statusNote.GetKind())
	assert.Equal(t, "Review the pull request", statusNote.GetHeadline())
	assert.Equal(t, "Review PR #42", statusNote.GetCtaLabel())
	assert.Equal(t, "https://github.com/acme/app/pull/42", statusNote.GetCtaUrl())
	assert.False(t, statusNote.GetShowOnlyWhenWaiting())
	require.NotNil(t, statusNote.GetAutomation())
	assert.Equal(t, appID.String(), statusNote.GetAutomation().GetAppId())
	assert.Equal(t, runID.String(), statusNote.GetRunId())
	assert.NotNil(t, statusNote.GetUpdatedAt())
	assert.True(t, statusNotes[1].GetShowOnlyWhenWaiting())

	// No notes stored serializes as absent, not as an empty list.
	bare := &models.FactoryWorkOrder{ID: uuid.New()}
	assert.Empty(t, mustSerializeWorkOrder(t, nil, bare, nil, nil).GetStatusNotes())
}

func TestSerializeWorkOrder_Origin(t *testing.T) {
	url := "https://github.com/acme/payments/issues/12"
	label := "acme/payments#12"
	serialized := mustSerializeWorkOrder(t, nil, &models.FactoryWorkOrder{
		ID:          uuid.New(),
		OriginURL:   &url,
		OriginLabel: &label,
	}, nil, nil)

	require.NotNil(t, serialized.GetOrigin())
	assert.Equal(t, url, serialized.GetOrigin().GetUrl())
	assert.Equal(t, label, serialized.GetOrigin().GetLabel())
	assert.Nil(t, mustSerializeWorkOrder(t, nil, &models.FactoryWorkOrder{ID: uuid.New()}, nil, nil).GetOrigin())
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
	}, nil)

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
	}, []string{"anthropic/claude-sonnet-4-6"})

	require.NotNil(t, out.GetRun())
	assert.Equal(t, runID.String(), out.GetRun().GetId())
	assert.Equal(t, canvasID.String(), out.GetRun().GetAppId())
	assert.Equal(t, "Implement app", out.GetRun().GetAppName())
	assert.Equal(t, []string{"anthropic/claude-sonnet-4-6"}, out.GetModels())
}

func TestSerializeWorkOrder_IncludesUsageBreakdown(t *testing.T) {
	executionID := uuid.New()
	dispatches := []models.FactoryWorkOrderLineDispatchRecord{
		{
			FactoryWorkOrderLineDispatch: models.FactoryWorkOrderLineDispatch{
				ID:     uuid.New(),
				LineID: uuid.New(),
			},
			Executions: []models.FactoryWorkOrderExecutionRecord{
				{
					FactoryWorkOrderExecution: models.FactoryWorkOrderExecution{
						ID:       executionID,
						StepName: "implement",
						Status:   models.FactoryWorkOrderExecutionStatusFinished,
						Result:   models.CanvasRunResultPassed,
					},
				},
			},
		},
	}

	serialized, err := serializeWorkOrder(nil, &models.FactoryWorkOrder{ID: uuid.New()}, dispatches, nil, workOrderUsageView{
		Totals: models.UsageTotals{
			TotalTokens:     1_000,
			DurationSeconds: 90,
			CostMicros:      730_000,
		},
		ByModel: []models.UsageByModel{
			{Provider: "anthropic", Model: "claude-sonnet-4-6", TotalTokens: 1_000, CostMicros: 450_000},
		},
		ByMachineType: []models.UsageByMachineType{
			{MachineType: "e1-large-amd64", DurationSeconds: 90, CostMicros: 280_000},
		},
		ModelsByExecution: map[uuid.UUID][]string{
			executionID: {"anthropic/claude-sonnet-4-6"},
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, 1000, serialized.GetTotalTokens())
	assert.EqualValues(t, 73, serialized.GetTotalCostCents())
	require.Len(t, serialized.GetUsageByModel(), 1)
	assert.Equal(t, "anthropic", serialized.GetUsageByModel()[0].GetProvider())
	assert.Equal(t, "claude-sonnet-4-6", serialized.GetUsageByModel()[0].GetModel())
	assert.EqualValues(t, 1000, serialized.GetUsageByModel()[0].GetTotalTokens())
	assert.EqualValues(t, 45, serialized.GetUsageByModel()[0].GetCostCents())
	require.Len(t, serialized.GetUsageByMachineType(), 1)
	assert.Equal(t, "e1-large-amd64", serialized.GetUsageByMachineType()[0].GetMachineType())
	assert.EqualValues(t, 90, serialized.GetUsageByMachineType()[0].GetDurationSeconds())
	assert.EqualValues(t, 28, serialized.GetUsageByMachineType()[0].GetCostCents())
	require.Len(t, serialized.GetLineDispatches(), 1)
	require.Len(t, serialized.GetLineDispatches()[0].GetStepExecutions(), 1)
	assert.Equal(t, []string{"anthropic/claude-sonnet-4-6"}, serialized.GetLineDispatches()[0].GetStepExecutions()[0].GetModels())
}

func TestSerializeFactoryPullRequest_HidesMergeableDuringActiveMutationRun(t *testing.T) {
	now := time.Now()
	runID := uuid.New()
	openMergeable := &models.FactoryPullRequest{
		ID:                  uuid.New(),
		FactoryID:           uuid.New(),
		WorkOrderID:         uuid.New(),
		State:               models.FactoryPullRequestStateOpen,
		Mergeable:           true,
		ActiveMutationRunID: &runID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	hidden := serializeFactoryPullRequest(openMergeable, 1, nil, nil, nil)
	assert.False(t, hidden.GetMergeable())

	openMergeable.ActiveMutationRunID = nil
	shown := serializeFactoryPullRequest(openMergeable, 1, nil, nil, nil)
	assert.True(t, shown.GetMergeable())
}

func TestAllocateCostCents_GivesTruncatedRemaindersToTheHeader(t *testing.T) {
	cents := allocateCostCents([]int64{6_000, 6_000}, 1)
	assert.Equal(t, []int64{1, 0}, cents)
	assert.Equal(t, int64(1), cents[0]+cents[1])
}

func TestAllocateCostCents_KeepsExactCents(t *testing.T) {
	cents := allocateCostCents([]int64{450_000, 280_000}, 73)
	assert.Equal(t, []int64{45, 28}, cents)
}

func TestSerializeWorkOrder_ReconcilesSubCentBreakdownToHeader(t *testing.T) {
	serialized, err := serializeWorkOrder(nil, &models.FactoryWorkOrder{ID: uuid.New()}, nil, nil, workOrderUsageView{
		Totals: models.UsageTotals{CostMicros: 12_000},
		ByModel: []models.UsageByModel{
			{Provider: "anthropic", Model: "claude-sonnet-4-6", TotalTokens: 10, CostMicros: 6_000},
			{Provider: "openai", Model: "gpt-4.1", TotalTokens: 10, CostMicros: 6_000},
		},
	})
	require.NoError(t, err)

	assert.EqualValues(t, 1, serialized.GetTotalCostCents())
	require.Len(t, serialized.GetUsageByModel(), 2)
	assert.EqualValues(t, 1, serialized.GetUsageByModel()[0].GetCostCents()+serialized.GetUsageByModel()[1].GetCostCents())
}

func TestSerializeFactory_IncludesPlanningDefaults(t *testing.T) {
	factory := &models.Factory{
		ID:                 uuid.New(),
		Name:               "Payments",
		Key:                "PAY",
		PlanningEnabled:    true,
		PlanningClarity:    true,
		PlanningConfidence: false,
	}

	serialized := serializeFactory(factory)
	require.NotNil(t, serialized.Planning)
	assert.True(t, serialized.Planning.Enabled)
	assert.True(t, serialized.Planning.Clarity)
	assert.False(t, serialized.Planning.Confidence)
}
