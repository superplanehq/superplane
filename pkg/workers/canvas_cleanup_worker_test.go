package workers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__CanvasCleanupWorker_ProcessesDeletedWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker()

	//
	// Create a canvas with nodes, events, executions, and queue items
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	// Create associated data
	event1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	event2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event1.ID, event2.ID, nil)
	support.CreateQueueItem(t, canvas.ID, "node-1", event1.ID, event2.ID)

	// Create canvas node execution KV
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(
		database.Conn(),
		canvas.ID,
		"node-1",
		execution.ID,
		"test-key",
		"test-value",
	))

	// Create workflow node request
	nodeRequest := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     "node-1",
		Type:       models.NodeRequestTypeInvokeAction,
		State:      models.NodeExecutionRequestStatePending,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "test",
				Parameters: map[string]any{},
			},
		}),
	}
	require.NoError(t, database.Conn().Create(&nodeRequest).Error)

	//
	// Verify all data exists before soft delete
	//
	_, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	support.VerifyCanvasEventsCount(t, canvas.ID, 2)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
	support.VerifyNodeQueueCount(t, canvas.ID, 1)

	// Verify KV and request exist
	support.VerifyNodeExecutionKVCount(t, canvas.ID, 1)
	support.VerifyNodeRequestCount(t, canvas.ID, 1)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err = canvas.SoftDelete()
	require.NoError(t, err)

	// Verify workflow is soft deleted
	_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Process the deleted canvas with cleanup worker
	// The worker now processes resources in batches, so it might take multiple calls
	//

	// Process until everything is cleaned up (with a reasonable limit)
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		err = worker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		// Check if workflow is completely deleted
		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	//
	// Verify everything is permanently deleted
	//

	// Canvas should be permanently deleted
	var canvasCount int64
	database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
	assert.Equal(t, int64(0), canvasCount)

	// All associated data should be permanently deleted
	nodes, err = models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 0)

	support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 0)
	support.VerifyNodeQueueCount(t, canvas.ID, 0)

	// KV and request should be deleted
	support.VerifyNodeExecutionKVCount(t, canvas.ID, 0)
	support.VerifyNodeRequestCount(t, canvas.ID, 0)
}

func Test__CanvasCleanupWorker_ProcessesWorkflowWithWebhook(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker()

	//
	// Create webhook
	//
	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStatePending,
		Secret: []byte("secret"),
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	//
	// Create a canvas with node that has webhook
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
				WebhookID: &webhookID,
			},
		},
		[]models.Edge{},
	)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err := canvas.SoftDelete()
	require.NoError(t, err)

	//
	// Verify webhook exists before cleanup
	//
	_, err = models.FindWebhook(webhookID)
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Process the deleted canvas with cleanup worker
	// May take multiple calls due to batched resource deletion
	//
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		err = worker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		// Check if workflow is completely deleted
		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	//
	// Verify webhook is soft deleted (marked for cleanup by webhook cleanup worker)
	//
	var webhookInDb models.Webhook
	err = database.Conn().Unscoped().Where("id = ?", webhookID).First(&webhookInDb).Error
	require.NoError(t, err)
	assert.NotNil(t, webhookInDb.DeletedAt)
}

func Test__CanvasCleanupWorker_HandlesEmptyWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker()

	//
	// Create a minimal canvas with no nodes, events, etc.
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err := canvas.SoftDelete()
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Process the deleted canvas with cleanup worker
	//
	err = worker.LockAndProcessCanvas(*deletedCanvas)
	require.NoError(t, err)

	//
	// Verify canvas is permanently deleted
	//
	var canvasCount int64
	database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
	assert.Equal(t, int64(0), canvasCount)
}

func Test__CanvasCleanupWorker_HandlesConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create a canvas with some data
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID, nil)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err := canvas.SoftDelete()
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Have two workers try to process the same canvas concurrently
	//
	results := make(chan error, 2)

	go func() {
		worker1 := NewCanvasCleanupWorker()
		results <- worker1.LockAndProcessCanvas(*deletedCanvas)
	}()

	go func() {
		worker2 := NewCanvasCleanupWorker()
		results <- worker2.LockAndProcessCanvas(*deletedCanvas)
	}()

	// Collect results - both should succeed (return nil)
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	// Process remaining work until fully cleaned up
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		worker := NewCanvasCleanupWorker()
		err := worker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		// Check if workflow is completely deleted
		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	//
	// Verify canvas is permanently deleted
	//
	var canvasCount int64
	database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
	assert.Equal(t, int64(0), canvasCount)

	// Verify associated data is cleaned up
	support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 0)
}

func Test__CanvasCleanupWorker_IgnoresNonDeletedWorkflows(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker()

	//
	// Create a normal (non-deleted) canvas
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	_ = support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	//
	// Try to process a non-deleted canvas (should be harmless)
	//
	err := worker.LockAndProcessCanvas(*canvas)
	require.NoError(t, err)

	//
	// Verify canvas and data still exist
	//
	_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
}
