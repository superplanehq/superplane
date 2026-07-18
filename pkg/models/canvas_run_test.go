package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__DeleteCanvasRunChains_DeletesFullRunData(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	childEvent := support.EmitCanvasEventForNode(t, run.WorkflowID, "node-1", "default", &execution.ID)
	require.NoError(t, database.Conn().Model(childEvent).Update("state", models.CanvasEventStateRouted).Error)

	queueItem := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	require.NoError(t, models.CreateNodeExecutionKVInTransaction(database.Conn(), run.WorkflowID, "node-1", execution.ID, "test-key", "test-value"))

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  run.WorkflowID,
		NodeID:      "node-1",
		ExecutionID: &execution.ID,
		State:       models.NodeExecutionRequestStateCompleted,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{ActionName: "test", Parameters: map[string]any{}},
		}),
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.DeleteCanvasRunChains(tx, []uuid.UUID{run.ID})
	}))

	var runCount int64
	require.NoError(t, database.Conn().Model(&models.CanvasRun{}).Where("id = ?", run.ID).Count(&runCount).Error)
	require.Equal(t, int64(0), runCount)

	support.VerifyCanvasEventsCount(t, run.WorkflowID, 0)
	support.VerifyNodeExecutionsCount(t, run.WorkflowID, 0)
	support.VerifyNodeQueueCount(t, run.WorkflowID, 0)
	support.VerifyNodeExecutionKVCount(t, run.WorkflowID, 0)
	support.VerifyNodeRequestCount(t, run.WorkflowID, 0)
}

func Test__CanvasRun__FindOpenWork__PendingOutputEvent(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	_, err := execution.Pass(map[string][]any{
		"default": {map[string]any{"data": "output"}},
	})
	require.NoError(t, err)

	openWork, err := run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.True(t, openWork.HasPendingEvents)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasQueueItems)

	var outputEvent models.CanvasEvent
	require.NoError(t, database.Conn().
		Where("execution_id = ?", execution.ID).
		First(&outputEvent).
		Error)
	assert.Equal(t, run.ID, outputEvent.RunID)

	require.NoError(t, outputEvent.Routed())
	openWork, err = run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.False(t, openWork.HasPendingEvents)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasQueueItems)
}

func Test__CanvasRun__FindOpenWork__QueueItem(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	queueItem := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	openWork, err := run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.True(t, openWork.HasQueueItems)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasPendingEvents)

	require.NoError(t, queueItem.Delete(database.Conn()))
	openWork, err = run.FindOpenWork(database.DB(t.Context()))
	require.NoError(t, err)
	assert.False(t, openWork.HasQueueItems)
	assert.False(t, openWork.HasActiveExecutions)
	assert.False(t, openWork.HasPendingEvents)
}

func Test__CanvasRun__FindOpenWork__ActiveExecution(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	for _, state := range []string{
		models.CanvasNodeExecutionStatePending,
		models.CanvasNodeExecutionStateStarted,
	} {
		require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
			"state":      state,
			"updated_at": time.Now(),
		}).Error)

		openWork, err := run.FindOpenWork(database.DB(t.Context()))
		require.NoError(t, err)
		assert.True(t, openWork.HasActiveExecutions, "expected active execution for state %s", state)
		assert.False(t, openWork.HasQueueItems)
		assert.False(t, openWork.HasPendingEvents)
	}
}

func Test__CanvasRun__CalculateResult__FailedTakesPrecedenceOverCancelled(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultCancelled,
		"updated_at": time.Now(),
	}).Error)

	failedExecution := createExecutionForRun(t, run, execution.RootEventID, "node-2")
	require.NoError(t, database.Conn().Model(failedExecution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultFailed,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultFailed, result)
}

func Test__CanvasRun__CalculateResult__Cancelled(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultCancelled,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultCancelled, result)
}

func Test__CanvasRun__CalculateResult__Passed(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	result, err := run.CalculateResult(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)
}

func Test__ListCanvasRuns__StatesAndResultsAreOredWithinWorkflow(t *testing.T) {
	r := support.Setup(t)

	canvasA, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)
	canvasB, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	runningA := createRunWithState(t, canvasA.ID, models.CanvasRunStateStarted, "")
	passedA := createRunWithState(t, canvasA.ID, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
	createRunWithState(t, canvasA.ID, models.CanvasRunStateFinished, models.CanvasRunResultFailed)

	// Runs in workflow B should never leak in even when their result matches the filter.
	createRunWithState(t, canvasB.ID, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
	createRunWithState(t, canvasB.ID, models.CanvasRunStateStarted, "")

	runs, err := models.ListCanvasRuns(canvasA.ID, 50, nil, models.CanvasRunFilters{
		States:  []string{models.CanvasRunStateStarted},
		Results: []string{models.CanvasRunResultPassed},
	})
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool, len(runs))
	for _, run := range runs {
		assert.Equal(t, canvasA.ID, run.WorkflowID, "filter must not leak runs from other workflows")
		ids[run.ID] = true
	}
	assert.True(t, ids[runningA.ID], "expected the running run to be included")
	assert.True(t, ids[passedA.ID], "expected the passed run to be included")
	assert.Len(t, runs, 2, "expected only running+passed runs from canvas A")
}

func createRunWithState(t *testing.T, workflowID uuid.UUID, state, result string) *models.CanvasRun {
	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		VersionID:  liveVersion.ID,
		State:      state,
		Result:     result,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	if state == models.CanvasRunStateFinished {
		run.FinishedAt = &now
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func setupRunWithExecution(t *testing.T) (*models.CanvasRun, *models.CanvasNodeExecution) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Type: models.NodeTypeComponent},
			{NodeID: "node-2", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)
	execution := createExecutionForRun(t, run, rootEvent.ID, "node-1")
	return run, execution
}

func createRunForRootEvent(t *testing.T, rootEvent *models.CanvasEvent) *models.CanvasRun {
	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}

		return rootEvent.RoutedInTransaction(tx)
	}))

	return run
}

func createExecutionForRun(t *testing.T, run *models.CanvasRun, rootEventID uuid.UUID, nodeID string) *models.CanvasNodeExecution {
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       rootEventID,
		State:         models.CanvasNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}
