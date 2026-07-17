package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DeleteCanvasNodeWithResult__RequestsCancellationForActiveExecutions(t *testing.T) {
	r := support.Setup(t)

	nodeID := "node-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: nodeID, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, nodeID, rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, nodeID)
	require.NoError(t, err)

	result, err := models.DeleteCanvasNodeWithResult(database.Conn(), *node)
	require.NoError(t, err)
	require.Contains(t, result.CancelledExecutionIDs, execution.ID)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

func Test__DeleteCanvasNodeWithResult__DeletesQueueItemsAndRequestsFinalization(t *testing.T) {
	r := support.Setup(t)

	nodeID := "node-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: nodeID, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	require.NoError(t, event.Routed())
	queueItem := support.CreateQueueItem(t, canvas.ID, nodeID, event.ID, event.ID)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, nodeID)
	require.NoError(t, err)

	result, err := models.DeleteCanvasNodeWithResult(database.Conn(), *node)
	require.NoError(t, err)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, nodeID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)

	run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, event.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, run.State)
	require.Len(t, result.DeletedQueueItems, 1)
	assert.Equal(t, queueItem.ID, result.DeletedQueueItems[0].ID)
	assert.Equal(t, event.RunID, result.DeletedQueueItems[0].RunID)
	assert.Empty(t, result.CancelledExecutionIDs)
}
