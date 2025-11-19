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

func Test__WorkflowCleanupWorker_ProcessesDeletedWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowCleanupWorker()

	//
	// Create a workflow with nodes, events, executions, and queue items
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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
	event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)
	execution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", event1.ID, event2.ID, nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, "node-1", event1.ID, event2.ID)

	// Create workflow node execution KV
	require.NoError(t, models.CreateWorkflowNodeExecutionKVInTransaction(
		database.Conn(),
		workflow.ID,
		"node-1",
		execution.ID,
		"test-key",
		"test-value",
	))

	// Create workflow node request
	nodeRequest := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
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
	_, err := models.FindWorkflow(r.Organization.ID, workflow.ID)
	require.NoError(t, err)
	nodes, err := models.FindWorkflowNodes(workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	support.VerifyWorkflowEventsCount(t, workflow.ID, 2)
	support.VerifyWorkflowNodeExecutionsCount(t, workflow.ID, 1)
	support.VerifyWorkflowNodeQueueCount(t, workflow.ID, 1)

	// Verify KV and request exist
	support.VerifyWorkflowNodeExecutionKVCount(t, workflow.ID, 1)
	support.VerifyWorkflowNodeRequestCount(t, workflow.ID, 1)

	//
	// Soft delete the workflow using the new soft delete method
	//
	err = workflow.SoftDelete()
	require.NoError(t, err)

	// Verify workflow is soft deleted
	_, err = models.FindWorkflow(r.Organization.ID, workflow.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedWorkflow, err := models.FindUnscopedWorkflow(workflow.ID)
	require.NoError(t, err)
	require.True(t, deletedWorkflow.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Process the deleted workflow with cleanup worker
	//
	err = worker.LockAndProcessWorkflow(*deletedWorkflow)
	require.NoError(t, err)

	//
	// Verify everything is permanently deleted
	//

	// Workflow should be permanently deleted
	var workflowCount int64
	database.Conn().Unscoped().Model(&models.Workflow{}).Where("id = ?", workflow.ID).Count(&workflowCount)
	assert.Equal(t, int64(0), workflowCount)

	// All associated data should be permanently deleted
	nodes, err = models.FindWorkflowNodes(workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 0)

	support.VerifyWorkflowEventsCount(t, workflow.ID, 0)
	support.VerifyWorkflowNodeExecutionsCount(t, workflow.ID, 0)
	support.VerifyWorkflowNodeQueueCount(t, workflow.ID, 0)

	// KV and request should be deleted
	support.VerifyWorkflowNodeExecutionKVCount(t, workflow.ID, 0)
	support.VerifyWorkflowNodeRequestCount(t, workflow.ID, 0)
}

func Test__WorkflowCleanupWorker_ProcessesWorkflowWithWebhook(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowCleanupWorker()

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
	// Create a workflow with node that has webhook
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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
	// Soft delete the workflow using the new soft delete method
	//
	err := workflow.SoftDelete()
	require.NoError(t, err)

	//
	// Verify webhook exists before cleanup
	//
	_, err = models.FindWebhook(webhookID)
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedWorkflow, err := models.FindUnscopedWorkflow(workflow.ID)
	require.NoError(t, err)
	require.True(t, deletedWorkflow.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Process the deleted workflow with cleanup worker
	//
	err = worker.LockAndProcessWorkflow(*deletedWorkflow)
	require.NoError(t, err)

	//
	// Verify webhook is soft deleted (marked for cleanup by webhook cleanup worker)
	//
	var webhookInDb models.Webhook
	err = database.Conn().Unscoped().Where("id = ?", webhookID).First(&webhookInDb).Error
	require.NoError(t, err)
	assert.NotNil(t, webhookInDb.DeletedAt)
}

func Test__WorkflowCleanupWorker_HandlesEmptyWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowCleanupWorker()

	//
	// Create a minimal workflow with no nodes, events, etc.
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	//
	// Soft delete the workflow using the new soft delete method
	//
	err := workflow.SoftDelete()
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedWorkflow, err := models.FindUnscopedWorkflow(workflow.ID)
	require.NoError(t, err)
	require.True(t, deletedWorkflow.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Process the deleted workflow with cleanup worker
	//
	err = worker.LockAndProcessWorkflow(*deletedWorkflow)
	require.NoError(t, err)

	//
	// Verify workflow is permanently deleted
	//
	var workflowCount int64
	database.Conn().Unscoped().Model(&models.Workflow{}).Where("id = ?", workflow.ID).Count(&workflowCount)
	assert.Equal(t, int64(0), workflowCount)
}

func Test__WorkflowCleanupWorker_HandlesConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create a workflow with some data
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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

	event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", event.ID, event.ID, nil)

	//
	// Soft delete the workflow using the new soft delete method
	//
	err := workflow.SoftDelete()
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedWorkflow, err := models.FindUnscopedWorkflow(workflow.ID)
	require.NoError(t, err)
	require.True(t, deletedWorkflow.DeletedAt.Valid, "DeletedAt should be set")

	//
	// Have two workers try to process the same workflow concurrently
	//
	results := make(chan error, 2)

	go func() {
		worker1 := NewWorkflowCleanupWorker()
		results <- worker1.LockAndProcessWorkflow(*deletedWorkflow)
	}()

	go func() {
		worker2 := NewWorkflowCleanupWorker()
		results <- worker2.LockAndProcessWorkflow(*deletedWorkflow)
	}()

	// Collect results - both should succeed (return nil)
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	//
	// Verify workflow is permanently deleted (only processed once)
	//
	var workflowCount int64
	database.Conn().Unscoped().Model(&models.Workflow{}).Where("id = ?", workflow.ID).Count(&workflowCount)
	assert.Equal(t, int64(0), workflowCount)

	// Verify associated data is cleaned up
	support.VerifyWorkflowEventsCount(t, workflow.ID, 0)
	support.VerifyWorkflowNodeExecutionsCount(t, workflow.ID, 0)
}

func Test__WorkflowCleanupWorker_IgnoresNonDeletedWorkflows(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewWorkflowCleanupWorker()

	//
	// Create a normal (non-deleted) workflow
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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

	_ = support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

	//
	// Try to process a non-deleted workflow (should be harmless)
	//
	err := worker.LockAndProcessWorkflow(*workflow)
	require.NoError(t, err)

	//
	// Verify workflow and data still exist
	//
	_, err = models.FindWorkflow(r.Organization.ID, workflow.ID)
	require.NoError(t, err)

	nodes, err := models.FindWorkflowNodes(workflow.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)

	support.VerifyWorkflowEventsCount(t, workflow.ID, 1)
}
