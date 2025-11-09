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
)

func Test__NodeRequestWorker_InvokeTriggerAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Registry)

	//
	// Create a simple workflow with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type": "daily",
					"time": "12:00",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking the emitEvent action on the schedule trigger.
	//
	request := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it completes successfully.
	//
	err := worker.LockAndProcessRequest(request)
	require.NoError(t, err)

	//
	// Verify the request was marked as completed.
	//
	var updatedRequest models.WorkflowNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestWorker_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create a simple workflow with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type": "daily",
					"time": "12:00",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a trigger action.
	//
	request := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Have two workers call LockAndProcessRequest concurrently on the same request.
	// LockAndProcessRequest uses a transaction with locking, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the request concurrently.
	//
	go func() {
		worker1 := NewNodeRequestWorker(r.Registry)
		results <- worker1.LockAndProcessRequest(request)
	}()

	go func() {
		worker2 := NewNodeRequestWorker(r.Registry)
		results <- worker2.LockAndProcessRequest(request)
	}()

	// Collect results - both should succeed (return nil)
	// because LockAndProcessRequest returns nil when it can't acquire the lock
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	//
	// Verify the request was marked as completed only once.
	//
	var updatedRequest models.WorkflowNodeRequest
	err := database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestWorker_UnsupportedRequestType(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Registry)

	//
	// Create a simple workflow with a trigger node.
	//
	triggerNode := "trigger-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type": "daily",
					"time": "12:00",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request with an unsupported type.
	//
	request := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		NodeID:     triggerNode,
		Type:       "unsupported-type",
		Spec:       datatypes.NewJSONType(models.NodeExecutionRequestSpec{}),
		State:      models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported node execution request type")
}

func Test__NodeRequestWorker_MissingInvokeActionSpec(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Registry)

	//
	// Create a simple workflow with a trigger node.
	//
	triggerNode := "trigger-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type": "daily",
					"time": "12:00",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request without an InvokeAction spec.
	//
	request := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec:       datatypes.NewJSONType(models.NodeExecutionRequestSpec{}), // Missing InvokeAction
		State:      models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec is not specified")
}

func Test__NodeRequestWorker_NonExistentTrigger(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Registry)

	//
	// Create a simple workflow with a trigger node that references a non-existent trigger.
	//
	triggerNode := "trigger-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "non-existent-trigger"}}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a trigger action.
	//
	request := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger not found")
}

func Test__NodeRequestWorker_NonExistentAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewNodeRequestWorker(r.Registry)

	//
	// Create a simple workflow with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type": "daily",
					"time": "12:00",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a non-existent action.
	//
	request := models.WorkflowNodeRequest{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "non-existent-action",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := worker.LockAndProcessRequest(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action 'non-existent-action' not found")
}
