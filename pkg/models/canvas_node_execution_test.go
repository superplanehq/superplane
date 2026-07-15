package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
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
