package models_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/test/support"
)

func Test__WorkspaceUsageEvent__TableName(t *testing.T) {
	assert.Equal(t, "workspace_usage_events", models.WorkspaceUsageEvent{}.TableName())
}

func Test__UsageTotalsAdd(t *testing.T) {
	combined := models.UsageTotals{TotalTokens: 10, CostMicros: 30_000}.Add(
		models.UsageTotals{DurationSeconds: 90, CostMicros: 50_040},
	)
	assert.Equal(t, int64(10), combined.TotalTokens)
	assert.Equal(t, int64(90), combined.DurationSeconds)
	assert.Equal(t, int64(80_040), combined.CostMicros)
	assert.Equal(t, int64(8), combined.CostCents())
}

func Test__RecordUsage__FactoryLinkedRunPersistsAndRollsUp(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	nodeExecutionID := uuid.New()

	err := models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: nodeExecutionID,
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		OutputTokens:    0,
		TotalTokens:     1_000_000,
	})
	require.NoError(t, err)

	assertInProgressExecutionUsage(t, db, execution.ID, 1_000_000, 300)

	totals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		FactoryID:      &execution.FactoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), totals.TotalTokens)
	assert.Equal(t, int64(300), totals.CostCents())
	require.Len(t, byModel, 1)
	assert.Equal(t, "anthropic", byModel[0].Provider)
	assert.Equal(t, "claude-sonnet-4-6", byModel[0].Model)
}

func Test__RecordUsage__SameNodeExecutionRecordsEachBilledCall(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	nodeExecutionID := uuid.New()

	first := models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: nodeExecutionID,
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}
	require.NoError(t, models.RecordUsage(db, first))
	require.NoError(t, models.RecordUsage(db, first))

	assertInProgressExecutionUsage(t, db, execution.ID, 2_000_000, 600)
}

func Test__RecordUsage__ConcurrentCallsKeepFullSpend(t *testing.T) {
	r := support.Setup(t)
	execution := dispatchWorkOrderExecution(t, r)

	const calls = 20
	var wg sync.WaitGroup
	errs := make(chan error, calls)
	for range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- models.RecordUsage(database.Conn(), models.WorkspaceUsageEventInput{
				OrganizationID:  r.Organization.ID,
				CanvasRunID:     requireExecutionRunID(t, execution),
				NodeExecutionID: uuid.New(),
				NodeID:          "prompt",
				Provider:        models.UsageProviderAnthropic,
				Model:           "claude-sonnet-4-6",
				InputTokens:     1_000_000,
				TotalTokens:     1_000_000,
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	assertInProgressExecutionUsage(t, database.Conn(), execution.ID, int64(calls)*1_000_000, int64(calls)*300)
}

func Test__RecordUsage__ChildRunUsesParentFactoryExecution(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	parentRun, err := models.FindUnscopedCanvasRun(db, requireExecutionRunID(t, execution))
	require.NoError(t, err)

	now := parentRun.CreatedAt
	if now == nil {
		created := time.Now()
		now = &created
	}
	childRun := models.CanvasRun{
		ID:               uuid.New(),
		WorkflowID:       parentRun.WorkflowID,
		NodeID:           parentRun.NodeID,
		VersionID:        parentRun.VersionID,
		ParentRunID:      &parentRun.ID,
		ParentWorkflowID: &parentRun.WorkflowID,
		State:            models.CanvasRunStateStarted,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(&childRun).Error)

	err = models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     childRun.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     100,
		OutputTokens:    20,
		TotalTokens:     120,
	})
	require.NoError(t, err)

	assertInProgressExecutionUsage(t, db, execution.ID, 120, 0)
}

func Test__RecordUsage__SkipsRunsWithoutFactoryCanvas(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)

	err = models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     100,
		OutputTokens:    20,
		TotalTokens:     120,
	})
	require.NoError(t, err)

	totals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), totals.TotalTokens)
	assert.Empty(t, byModel)
}

func Test__RecordUsage__FactoryCanvasWithoutLineStepPersists(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	run := startFactoryCanvasRun(t, r, factory.ID, nil)

	require.NoError(t, models.RecordUsage(db, sonnetUsage(t, r, run.ID)))

	event := requireUsageEventForRun(t, db, run.ID)
	assert.Equal(t, factory.ID, *event.FactoryID)
	assert.Nil(t, event.WorkOrderID)
	assert.Nil(t, event.WorkOrderExecutionID)
	assert.Equal(t, int64(1_000_000), event.TotalTokens)
}

func Test__RecordUsage__AttachesWorkOrderFromPullRequestLink(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order := createFactoryOrder(t, r)
	run := startFactoryCanvasRun(t, r, factory.ID, nil)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/99",
	})
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, run.ID, "Please add tests."))

	require.NoError(t, models.RecordUsage(db, sonnetUsage(t, r, run.ID)))

	event := requireUsageEventForRun(t, db, run.ID)
	require.NotNil(t, event.WorkOrderID)
	assert.Equal(t, order.ID, *event.WorkOrderID)
	assert.Nil(t, event.WorkOrderExecutionID)
}

func Test__RecordUsage__ChildRunInheritsPullRequestWorkOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order := createFactoryOrder(t, r)
	parent := startFactoryCanvasRun(t, r, factory.ID, nil)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/100",
	})
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, parent.ID, "Address review."))

	now := parent.CreatedAt
	if now == nil {
		created := time.Now()
		now = &created
	}
	child := models.CanvasRun{
		ID:               uuid.New(),
		WorkflowID:       parent.WorkflowID,
		NodeID:           parent.NodeID,
		VersionID:        parent.VersionID,
		ParentRunID:      &parent.ID,
		ParentWorkflowID: &parent.WorkflowID,
		State:            models.CanvasRunStateStarted,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(&child).Error)

	require.NoError(t, models.RecordUsage(db, sonnetUsage(t, r, child.ID)))

	event := requireUsageEventForRun(t, db, child.ID)
	require.NotNil(t, event.WorkOrderID)
	assert.Equal(t, order.ID, *event.WorkOrderID)
}

func Test__RecordUsage__AttachesWorkOrderFromOnWorkOrderEvent(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order := createFactoryOrder(t, r)
	run := startFactoryCanvasRun(t, r, factory.ID, map[string]any{
		"type": "workOrder.created",
		"data": map[string]any{
			"workOrder": map[string]any{"id": order.ID.String()},
		},
	})

	require.NoError(t, models.RecordUsage(db, sonnetUsage(t, r, run.ID)))

	event := requireUsageEventForRun(t, db, run.ID)
	require.NotNil(t, event.WorkOrderID)
	assert.Equal(t, order.ID, *event.WorkOrderID)
	assert.Nil(t, event.WorkOrderExecutionID)
}

func Test__SumUsageForRunTrees__IncludesDescendantSpend(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	parent := startFactoryCanvasRun(t, r, factory.ID, nil)
	now := parent.CreatedAt
	if now == nil {
		created := time.Now()
		now = &created
	}
	child := models.CanvasRun{
		ID:               uuid.New(),
		WorkflowID:       parent.WorkflowID,
		NodeID:           parent.NodeID,
		VersionID:        parent.VersionID,
		ParentRunID:      &parent.ID,
		ParentWorkflowID: &parent.WorkflowID,
		State:            models.CanvasRunStateStarted,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(&child).Error)
	require.NoError(t, models.RecordUsage(db, sonnetUsage(t, r, child.ID)))

	sums, err := models.SumUsageForRunTrees(db, []uuid.UUID{parent.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), sums[parent.ID].TotalTokens)
	assert.Equal(t, int64(300), sums[parent.ID].CostCents())
}

func Test__SumUsageForWorkOrders__SumsLedgerByWorkOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order := createFactoryOrder(t, r)
	run := startFactoryCanvasRun(t, r, factory.ID, map[string]any{
		"type": "workOrder.created",
		"data": map[string]any{
			"workOrder": map[string]any{"id": order.ID.String()},
		},
	})
	require.NoError(t, models.RecordUsage(db, sonnetUsage(t, r, run.ID)))

	sums, err := models.SumUsageForWorkOrders(db, []uuid.UUID{order.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), sums[order.ID].TotalTokens)
	assert.Equal(t, int64(300), sums[order.ID].CostCents())
}

func Test__RecordUsage__SameIdempotencyKeyRecordsOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	nodeExecutionID := uuid.New()
	key := models.UsageIdempotencyKeyRunner + ":" + nodeExecutionID.String()

	input := models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: nodeExecutionID,
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		IdempotencyKey:  key,
	}
	require.NoError(t, models.RecordUsage(db, input))
	require.NoError(t, models.RecordUsage(db, input))

	var count int64
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).Where("idempotency_key = ?", key).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assertInProgressExecutionUsage(t, db, execution.ID, 1_000_000, 300)
}

func Test__RecordUsage__HostedUnknownModelFailsClosed(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	err := models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenRouter,
		Model:           "unknown-hosted-model",
		InputTokens:     100,
		TotalTokens:     100,
		FundingSource:   models.UsageFundingSourceHosted,
	})
	require.ErrorIs(t, err, models.ErrHostedUsageUnpriced)
}

func assertInProgressExecutionUsage(t *testing.T, db *gorm.DB, executionID uuid.UUID, tokens, cents int64) {
	t.Helper()

	var updated models.FactoryWorkOrderExecution
	require.NoError(t, db.First(&updated, "id = ?", executionID).Error)
	assert.NotEqual(t, models.FactoryWorkOrderExecutionStatusFinished, updated.Status)
	assert.Equal(t, tokens, updated.TotalTokens)
	assert.Equal(t, cents, updated.CostCents)
}

func requireExecutionRunID(t *testing.T, execution *models.FactoryWorkOrderExecution) uuid.UUID {
	t.Helper()
	require.NotNil(t, execution.RunID)
	return *execution.RunID
}

func dispatchWorkOrderExecution(t *testing.T, r *support.ResourceRegistry) *models.FactoryWorkOrderExecution {
	t.Helper()
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
	}))

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
	return execution
}

func createFactoryOrder(t *testing.T, r *support.ResourceRegistry) (*models.Factory, *models.FactoryWorkOrder) {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	return factory, order
}

func startFactoryCanvasRun(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID, data map[string]any) *models.CanvasRun {
	t.Helper()
	db := database.DB(t.Context())
	canvas, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryID, "score", "start")
	if data == nil {
		data = map[string]any{"key": "value"}
	}
	event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, entry, "default", nil, data)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)
	return run
}

func sonnetUsage(t *testing.T, r *support.ResourceRegistry, runID uuid.UUID) models.WorkspaceUsageEventInput {
	t.Helper()
	return models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}
}

func requireUsageEventForRun(t *testing.T, db *gorm.DB, runID uuid.UUID) models.WorkspaceUsageEvent {
	t.Helper()
	var event models.WorkspaceUsageEvent
	require.NoError(t, db.Where("canvas_run_id = ?", runID).First(&event).Error)
	return event
}

func Test__RecordComputeUsage__FactoryLinkedRunPersistsAndRollsUp(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	err := models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 12,
		IdempotencyKey:  "runner:compute:task-1",
	})
	require.NoError(t, err)

	var updated models.FactoryWorkOrderExecution
	require.NoError(t, db.First(&updated, "id = ?", execution.ID).Error)
	assert.Equal(t, int64(0), updated.TotalTokens)
	assert.Equal(t, int64(0), updated.CostCents)
	assert.Equal(t, int64(12), updated.DurationSeconds)

	modelTotals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		FactoryID:      &execution.FactoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), modelTotals.TotalTokens)
	assert.Empty(t, byModel)

	computeTotals, byMachine, err := models.SummarizeComputeUsage(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		FactoryID:      &execution.FactoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(12), computeTotals.DurationSeconds)
	require.Len(t, byMachine, 1)
	assert.Equal(t, "e1-large-amd64", byMachine[0].MachineType)
	assert.Equal(t, int64(12), byMachine[0].DurationSeconds)
}

func Test__RecordComputeUsage__SameIdempotencyKeyRecordsOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	input := models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-tiny-arm64",
		FleetID:         "local",
		DurationSeconds: 3,
		IdempotencyKey:  "runner:compute:same-task",
	}
	require.NoError(t, models.RecordComputeUsage(db, input))
	require.NoError(t, models.RecordComputeUsage(db, input))

	var count int64
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).Where("idempotency_key = ?", input.IdempotencyKey).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func Test__RecordComputeUsage__SkipsRunsWithoutFactoryCanvas(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)

	err = models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		DurationSeconds: 9,
		IdempotencyKey:  "runner:compute:org-canvas",
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).Where("canvas_run_id = ?", run.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func Test__RecordComputeUsage__LocalFleetCostIsZero(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	t.Cleanup(pricebook.Reset)
	pricebook.SetComputeRates(map[string]int64{"e1-large-amd64": 100})

	err := models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "local",
		DurationSeconds: 10,
	})
	require.NoError(t, err)

	event := requireUsageEventForRun(t, db, requireExecutionRunID(t, execution))
	assert.Equal(t, models.UsageKindCompute, event.UsageKind)
	assert.Equal(t, "local", event.FleetID)
	assert.Equal(t, int64(0), event.CostMicros)
}

func Test__RecordComputeUsage__WritesCatalogCost(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	t.Cleanup(pricebook.Reset)

	err := models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 3600,
		IdempotencyKey:  "runner:compute:priced-task",
	})
	require.NoError(t, err)

	event := requireUsageEventForRun(t, db, requireExecutionRunID(t, execution))
	assert.Equal(t, int64(3600)*pricebook.MicrosPerSecondE1Large, event.CostMicros)

	var updated models.FactoryWorkOrderExecution
	require.NoError(t, db.First(&updated, "id = ?", execution.ID).Error)
	assert.Equal(t, int64(200), updated.CostCents)
	assert.Equal(t, int64(3600), updated.DurationSeconds)
}
