package workers

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__WorkflowNodeExecutor_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple workflow with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a pending execution for the component node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	execution := support.CreateWorkflowNodeExecution(t, workflow.ID, componentNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Have two workers call LockAndProcessNodeExecution concurrently on the same execution.
	// LockAndProcessNodeExecution uses a transaction with SKIP LOCKED, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the execution concurrently.
	//
	go func() {
		executor1 := NewWorkflowNodeExecutor(r.Registry)
		results <- executor1.LockAndProcessNodeExecution(*execution)
	}()

	go func() {
		executor2 := NewWorkflowNodeExecutor(r.Registry)
		results <- executor2.LockAndProcessNodeExecution(*execution)
	}()

	// Collect results - one should succeed (return nil) and one should get ErrRecordLocked
	// because LockAndProcessNodeExecution returns ErrRecordLocked when it can't acquire the lock
	result1 := <-results
	result2 := <-results

	successCount, lockedCount := countConcurrentExecutionResults(t, []error{result1, result2})
	assert.Equal(t, 1, successCount, "Exactly one worker should succeed")
	assert.Equal(t, 1, lockedCount, "Exactly one worker should get ErrRecordLocked")

	//
	// Verify the execution was started and finished (since noop completes immediately).
	// If both workers processed it, we would see inconsistent state or errors.
	//
	updatedExecution, err := models.FindNodeExecution(workflow.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultPassed, updatedExecution.Result)
}

func Test__WorkflowNodeExecutor_BlueprintNodeExecution(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple blueprint with a noop node
	//
	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "noop1",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
		},
		[]models.Edge{},
		[]models.BlueprintOutputChannel{
			{
				Name:              "default",
				NodeID:            "noop1",
				NodeOutputChannel: "default",
			},
		},
	)

	//
	// Create a workflow with a trigger and a blueprint node.
	//
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: blueprintNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a pending execution for the blueprint node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	execution := support.CreateWorkflowNodeExecution(t, workflow.ID, blueprintNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Process the execution and verify the blueprint node creates a child execution
	// and moves the parent execution to started state.
	//
	executor := NewWorkflowNodeExecutor(r.Registry)
	err := executor.LockAndProcessNodeExecution(*execution)
	require.NoError(t, err)

	// Verify parent execution moved to started state
	parentExecution, err := models.FindNodeExecution(workflow.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateStarted, parentExecution.State)

	// Verify child execution was created with pending state
	childExecutions, err := models.FindChildExecutions(execution.ID, []string{
		models.WorkflowNodeExecutionStatePending,
		models.WorkflowNodeExecutionStateStarted,
		models.WorkflowNodeExecutionStateFinished,
	})

	require.NoError(t, err)
	require.Len(t, childExecutions, 1)
	assert.Equal(t, models.WorkflowNodeExecutionStatePending, childExecutions[0].State)
	assert.Equal(t, rootEvent.ID, childExecutions[0].RootEventID)
	assert.Equal(t, &execution.ID, childExecutions[0].ParentExecutionID)
}

func Test__WorkflowNodeExecutor_ComponentNodeWithoutStateChange(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple workflow with a trigger and an approval component node.
	// The approval component does NOT change state on Execute() - it just sets metadata.
	//
	triggerNode := "trigger-1"
	approvalNode := "approval-1"
	approvalConfiguration := map[string]any{
		"items": []any{
			map[string]any{
				"type": "user",
				"user": r.User.String(),
			},
		},
	}

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID:        approvalNode,
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "approval"}}),
				Configuration: datatypes.NewJSONType(approvalConfiguration),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: approvalNode, Channel: "default"},
		},
	)

	nodes, err := models.FindWorkflowNodes(workflow.ID)
	require.NoError(t, err)

	log.Printf("nodes: %v", nodes)

	//
	// Create a root event and a pending execution for the approval node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, workflow.ID, approvalNode, rootEvent.ID, rootEvent.ID, nil, approvalConfiguration)

	//
	// Process the execution and verify the execution is started but NOT finished.
	// The approval component doesn't call Pass() in Execute(), so it should remain in started state.
	//
	executor := NewWorkflowNodeExecutor(r.Registry)
	err = executor.LockAndProcessNodeExecution(*execution)
	require.NoError(t, err)

	// Verify execution moved to started state but not finished,
	// and metadata is updated.
	updatedExecution, err := models.FindNodeExecution(workflow.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateStarted, updatedExecution.State)
	assert.Equal(t, "", updatedExecution.Result)
	assert.Equal(t, map[string]any{
		"result": "pending",
		"records": []any{
			map[string]any{
				"index": float64(0),
				"type":  "user",
				"state": "pending",
				"user": map[string]any{
					"id":    r.User.String(),
					"name":  r.UserModel.Name,
					"email": r.UserModel.Email,
				},
			},
		},
	}, updatedExecution.Metadata.Data())
}

func Test__WorkflowNodeExecutor_ComponentNodeWithStateChange(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple workflow with a trigger and a noop component node.
	// The noop component DOES change state on Execute() - it calls Pass() immediately.
	//
	triggerNode := "trigger-1"
	noopNode := "noop-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: noopNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: noopNode, Channel: "default"},
		},
	)

	//
	// Create a root event and a pending execution for the noop node.
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, triggerNode, "default", nil)
	execution := support.CreateWorkflowNodeExecution(t, workflow.ID, noopNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Process the execution and verify the execution is both started AND finished.
	// The noop component calls Pass() in Execute(), which should finish the execution.
	//
	executor := NewWorkflowNodeExecutor(r.Registry)
	err := executor.LockAndProcessNodeExecution(*execution)
	require.NoError(t, err)

	// Verify execution moved to finished state with passed result
	updatedExecution, err := models.FindNodeExecution(workflow.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.WorkflowNodeExecutionResultPassed, updatedExecution.Result)
}

func countConcurrentExecutionResults(t *testing.T, results []error) (successCount int, lockedCount int) {
	for i, result := range results {
		switch result {
		case nil:
			successCount++
		case ErrRecordLocked:
			lockedCount++
		default:
			t.Fatalf("Unexpected error from worker %d: %v", i+1, result)
		}
	}
	return successCount, lockedCount
}
