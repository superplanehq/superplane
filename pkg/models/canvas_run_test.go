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
