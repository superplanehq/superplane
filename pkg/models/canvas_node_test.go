package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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

	queueItems, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, nodeID, 10, nil)
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

func Test__DeleteCanvasNodeWithResult__SoftDeletesOrphanedWebhook(t *testing.T) {
	r := support.Setup(t)
	webhookID := createWebhook(t)
	_, node := createCanvasNodeWithWebhook(t, r, webhookID)

	_, err := models.DeleteCanvasNodeWithResult(database.Conn(), node)
	require.NoError(t, err)

	var stored models.Webhook
	require.NoError(t, database.Conn().Unscoped().First(&stored, webhookID).Error)
	assert.True(t, stored.DeletedAt.Valid)
}

func Test__DeleteCanvasNodeWithResult__PreservesSharedWebhookAcrossCanvases(t *testing.T) {
	r := support.Setup(t)
	webhookID := createWebhook(t)
	_, nodeA := createCanvasNodeWithWebhook(t, r, webhookID)
	_, _ = createCanvasNodeWithWebhook(t, r, webhookID)

	_, err := models.DeleteCanvasNodeWithResult(database.Conn(), nodeA)
	require.NoError(t, err)

	_, err = models.FindWebhook(webhookID)
	require.NoError(t, err)
}

func Test__ListRetiredTriggerNodes__IncludesRemovedIntakes(t *testing.T) {
	r := support.Setup(t)

	activeCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		datadogIntakeCanvasNode("active"),
	}, nil)
	deletedNodeCanvas, deletedNodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		datadogIntakeCanvasNode("removed"),
	}, nil)
	deletedCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		datadogIntakeCanvasNode("canvas"),
	}, nil)

	require.NoError(t, models.DeleteCanvasNode(database.Conn(), deletedNodes[0]))
	require.NoError(t, deletedCanvas.SoftDelete())

	nodes, err := models.ListRetiredTriggerNodes(database.Conn(), "datadog.onErrorTrackingAlert", "monitorId")
	require.NoError(t, err)

	ids := map[string]bool{}
	for _, node := range nodes {
		if node.WorkflowID == activeCanvas.ID || node.WorkflowID == deletedNodeCanvas.ID || node.WorkflowID == deletedCanvas.ID {
			ids[node.WorkflowID.String()+"/"+node.NodeID] = true
		}
	}
	assert.False(t, ids[activeCanvas.ID.String()+"/active"])
	assert.True(t, ids[deletedNodeCanvas.ID.String()+"/removed"])
	assert.True(t, ids[deletedCanvas.ID.String()+"/canvas"])
}

func datadogIntakeCanvasNode(nodeID string) models.CanvasNode {
	return models.CanvasNode{
		NodeID: nodeID,
		Name:   "Datadog",
		Type:   models.NodeTypeTrigger,
		Ref: datatypes.NewJSONType(models.NodeRef{
			Trigger: &models.TriggerRef{Name: "datadog.onErrorTrackingAlert"},
		}),
		Configuration: datatypes.NewJSONType(map[string]any{"service": "checkout"}),
		Metadata:      datatypes.NewJSONType(map[string]any{"monitorId": "9", "monitorService": "checkout"}),
	}
}
