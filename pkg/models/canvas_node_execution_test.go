package models_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__CanvasNodeExecution_PassCompletesPendingRequests(t *testing.T) {
	_, execution := setupRunWithExecution(t)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  execution.WorkflowID,
		NodeID:      execution.NodeID,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "poll",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
		RunAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	_, err := execution.Pass(map[string][]any{})
	require.NoError(t, err)

	pending, err := models.CountPendingRequestsForExecutionsInTransaction(database.Conn(), []uuid.UUID{execution.ID})
	require.NoError(t, err)
	assert.Zero(t, pending)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__CanvasNodeExecution_PassIsNoopAfterFinished(t *testing.T) {
	_, execution := setupRunWithExecution(t)

	_, err := execution.Pass(map[string][]any{"default": {map[string]any{"n": 1}}})
	require.NoError(t, err)

	finishedAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Update("updated_at", finishedAt).Error)
	execution.UpdatedAt = &finishedAt

	eventsBefore, err := execution.GetOutputs()
	require.NoError(t, err)

	events, err := execution.Pass(map[string][]any{"default": {map[string]any{"n": 2}}})
	require.NoError(t, err)
	assert.Empty(t, events)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second)

	eventsAfter, err := execution.GetOutputs()
	require.NoError(t, err)
	assert.Len(t, eventsAfter, len(eventsBefore))
}

func Test__CanvasNodeExecution_PassIsNoopWhenPersistedExecutionIsFinished(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	staleExecution := *execution

	finishedAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": finishedAt,
	}).Error)

	events, err := staleExecution.Pass(map[string][]any{"default": {map[string]any{"n": 1}}})
	require.NoError(t, err)
	assert.Empty(t, events)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second)

	eventsAfter, err := staleExecution.GetOutputs()
	require.NoError(t, err)
	assert.Empty(t, eventsAfter)
}

func Test__CanvasNodeExecution_PassRetriesAfterRolledBackTransaction(t *testing.T) {
	_, execution := setupRunWithExecution(t)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, err := execution.PassInTransaction(tx, map[string][]any{})
		require.NoError(t, err)
		return errors.New("force rollback")
	})
	require.Error(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State)

	var rolledBackExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&rolledBackExecution).Error)
	require.Equal(t, models.CanvasNodeExecutionStatePending, rolledBackExecution.State)

	_, err = execution.Pass(map[string][]any{})
	require.NoError(t, err)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
}

func Test__CanvasNodeExecution_PassIsNoopWhileCancelling(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	staleExecution := *execution

	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateCancelling,
	}).Error)

	events, err := staleExecution.Pass(map[string][]any{"default": {map[string]any{"n": 1}}})
	require.NoError(t, err)
	assert.Empty(t, events)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

func Test__CanvasNodeExecution_FailIsNoopWhileCancelling(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	staleExecution := *execution

	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateCancelling,
	}).Error)

	failed, err := staleExecution.FailInTransaction(database.Conn(), models.CanvasNodeExecutionResultReasonError, "too late")
	require.NoError(t, err)
	assert.False(t, failed)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

func setExecutionQueueName(t *testing.T, execution *models.CanvasNodeExecution, queueName string) {
	require.NoError(t, database.Conn().Model(execution).Update("queue_name", queueName).Error)
	execution.QueueName = &queueName
}

func Test__ListActiveNodeExecutionsInQueues_ReturnsMatchingActiveExecutions(t *testing.T) {
	run, execution := setupRunWithExecution(t)
	setExecutionQueueName(t, execution, "queue-a")

	otherNodeExecution := createExecutionForRun(t, run, execution.RootEventID, "node-2")
	setExecutionQueueName(t, otherNodeExecution, "queue-b")

	executions, err := models.ListActiveNodeExecutionsInQueues(
		database.Conn(),
		execution.WorkflowID,
		[]string{"node-1", "node-2"},
		[]string{"queue-a"},
	)

	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, execution.ID, executions[0].ID)
}

func Test__ListActiveNodeExecutionsInQueues_ExcludesFinishedExecutions(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	setExecutionQueueName(t, execution, "queue-a")

	_, err := execution.Pass(map[string][]any{})
	require.NoError(t, err)

	executions, err := models.ListActiveNodeExecutionsInQueues(
		database.Conn(),
		execution.WorkflowID,
		[]string{"node-1"},
		[]string{"queue-a"},
	)

	require.NoError(t, err)
	assert.Empty(t, executions)
}

func Test__ListActiveNodeExecutionsInQueues_ExcludesOtherWorkflows(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	setExecutionQueueName(t, execution, "queue-a")

	executions, err := models.ListActiveNodeExecutionsInQueues(
		database.Conn(),
		uuid.New(),
		[]string{"node-1"},
		[]string{"queue-a"},
	)

	require.NoError(t, err)
	assert.Empty(t, executions)
}

func Test__ListActiveNodeExecutionsInQueues_ReturnsEmptyWithoutQueryingForEmptyInputs(t *testing.T) {
	_, execution := setupRunWithExecution(t)
	setExecutionQueueName(t, execution, "queue-a")

	executions, err := models.ListActiveNodeExecutionsInQueues(database.Conn(), execution.WorkflowID, nil, []string{"queue-a"})
	require.NoError(t, err)
	assert.Empty(t, executions)

	executions, err = models.ListActiveNodeExecutionsInQueues(database.Conn(), execution.WorkflowID, []string{"node-1"}, nil)
	require.NoError(t, err)
	assert.Empty(t, executions)
}
