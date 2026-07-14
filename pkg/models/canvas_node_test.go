package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DeleteCanvasNodeWithResult__DeletesQueueItemsAndFinishesRun(t *testing.T) {
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
	support.CreateQueueItem(t, canvas.ID, nodeID, event.ID, event.ID)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, nodeID)
	require.NoError(t, err)

	result, err := models.DeleteCanvasNodeWithResult(database.Conn(), *node)
	require.NoError(t, err)

	queueItems, err := models.ListNodeQueueItems(canvas.ID, nodeID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)

	run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, event.RunID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, run.State)
	assert.Equal(t, models.CanvasRunResultPassed, run.Result)
	assert.Contains(t, result.FinishedRunIDs, event.RunID)
	assert.Empty(t, result.CancelledExecutionIDs)
}
