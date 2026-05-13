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

func Test__CanvasRun__PendingOutputEventKeepsRunStarted(t *testing.T) {
	run, execution := setupRunWithExecution(t)

	_, err := execution.Pass(map[string][]any{
		"default": {map[string]any{"data": "output"}},
	})
	require.NoError(t, err)

	updatedRun := findRun(t, run.ID)
	assert.Equal(t, models.CanvasRunStateStarted, updatedRun.State)
	assert.Empty(t, updatedRun.Result)
	assert.Nil(t, updatedRun.FinishedAt)

	var outputEvent models.CanvasEvent
	require.NoError(t, database.Conn().
		Where("execution_id = ?", execution.ID).
		First(&outputEvent).
		Error)
	assert.Equal(t, run.ID, outputEvent.RunID)

	require.NoError(t, outputEvent.Routed())
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.MaybeFinalizeRunInTransaction(tx, run.ID)
		return err
	}))

	updatedRun = findRun(t, run.ID)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
	assert.NotNil(t, updatedRun.FinishedAt)
}

func Test__CanvasRun__QueueItemKeepsRunStarted(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": time.Now(),
	}).Error)

	queueItem := support.CreateQueueItem(t, run.WorkflowID, "node-1", execution.RootEventID, execution.RootEventID)
	queueItem.RunID = run.ID
	require.NoError(t, database.Conn().Save(queueItem).Error)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.MaybeFinalizeRunInTransaction(tx, run.ID)
		return err
	}))

	updatedRun := findRun(t, run.ID)
	assert.Equal(t, models.CanvasRunStateStarted, updatedRun.State)

	require.NoError(t, queueItem.Delete(database.Conn()))
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.MaybeFinalizeRunInTransaction(tx, run.ID)
		return err
	}))

	updatedRun = findRun(t, run.ID)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
}

func Test__CanvasRun__ResultPrecedence(t *testing.T) {
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

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := models.MaybeFinalizeRunInTransaction(tx, run.ID)
		return err
	}))

	updatedRun := findRun(t, run.ID)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultFailed, updatedRun.Result)
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
	run := models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
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

func findRun(t *testing.T, runID uuid.UUID) *models.CanvasRun {
	var run models.CanvasRun
	require.NoError(t, database.Conn().
		Where("id = ?", runID).
		First(&run).
		Error)
	return &run
}
