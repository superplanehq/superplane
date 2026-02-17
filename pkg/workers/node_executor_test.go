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

func Test__NodeExecutor_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple canvas with a trigger and a component node.
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Have two workers call LockAndProcessNodeExecution concurrently on the same execution.
	// LockAndProcessNodeExecution uses a transaction with SKIP LOCKED, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the execution concurrently.
	//
	go func() {
		executor1 := NewNodeExecutor(r.Encryptor, r.Registry, "http://localhost", "http://localhost")
		results <- executor1.LockAndProcessNodeExecution(execution.ID)
	}()

	go func() {
		executor2 := NewNodeExecutor(r.Encryptor, r.Registry, "http://localhost", "http://localhost")
		results <- executor2.LockAndProcessNodeExecution(execution.ID)
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
	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
}

func Test__NodeExecutor_BlueprintNodeExecution(t *testing.T) {
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
	// Create a canvas with a trigger and a blueprint node.
	//
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, blueprintNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Process the execution and verify the blueprint node creates a child execution
	// and moves the parent execution to started state.
	//
	executor := NewNodeExecutor(r.Encryptor, r.Registry, "http://localhost", "http://localhost")
	err := executor.LockAndProcessNodeExecution(execution.ID)
	require.NoError(t, err)

	// Verify parent execution moved to started state
	parentExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateStarted, parentExecution.State)

	// Verify child execution was created with pending state
	childExecutions, err := models.FindChildExecutions(execution.ID, []string{
		models.CanvasNodeExecutionStatePending,
		models.CanvasNodeExecutionStateStarted,
		models.CanvasNodeExecutionStateFinished,
	})

	require.NoError(t, err)
	require.Len(t, childExecutions, 1)
	assert.Equal(t, models.CanvasNodeExecutionStatePending, childExecutions[0].State)
	assert.Equal(t, rootEvent.ID, childExecutions[0].RootEventID)
	assert.Equal(t, &execution.ID, childExecutions[0].ParentExecutionID)
}

func Test__NodeExecutor_ComponentNodeWithoutStateChange(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple canvas with a trigger and an approval component node.
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

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)

	log.Printf("nodes: %v", nodes)

	//
	// Create a root event and a pending execution for the approval node.
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, approvalNode, rootEvent.ID, rootEvent.ID, nil, approvalConfiguration)

	//
	// Process the execution and verify the execution is started but NOT finished.
	// The approval component doesn't call Pass() in Execute(), so it should remain in started state.
	//
	executor := NewNodeExecutor(r.Encryptor, r.Registry, "http://localhost", "http://localhost")
	err = executor.LockAndProcessNodeExecution(execution.ID)
	require.NoError(t, err)

	// Verify execution moved to started state but not finished,
	// and metadata is updated.
	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateStarted, updatedExecution.State)
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
					"email": r.UserModel.GetEmail(),
				},
			},
		},
	}, updatedExecution.Metadata.Data())
}

func Test__NodeExecutor_ComponentNodeWithStateChange(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a simple canvas with a trigger and a noop component node.
	// The noop component DOES change state on Execute() - it calls Pass() immediately.
	//
	triggerNode := "trigger-1"
	noopNode := "noop-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, noopNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Process the execution and verify the execution is both started AND finished.
	// The noop component calls Pass() in Execute(), which should finish the execution.
	//
	executor := NewNodeExecutor(r.Encryptor, r.Registry, "http://localhost", "http://localhost")
	err := executor.LockAndProcessNodeExecution(execution.ID)
	require.NoError(t, err)

	// Verify execution moved to finished state with passed result
	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
}

func Test__NodeExecutor_BlueprintNodeExecutionFailsWhenConfigurationCannotBeBuilt(t *testing.T) {
	r := support.Setup(t)

	//
	// Create a blueprint with a noop node that has invalid configuration.
	// The configuration references a variable that doesn't exist, which should
	// cause the configuration builder to fail.
	//
	invalidConfiguration := map[string]any{
		"invalid_field": "{{ .nonexistent_variable }}",
	}

	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:            "noop1",
				Type:          models.NodeTypeComponent,
				Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
				Configuration: invalidConfiguration,
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
	// Create a canvas with a trigger and a blueprint node.
	//
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, blueprintNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Process the execution and verify it fails due to configuration build error.
	// LockAndProcessNodeExecution should not return an error,
	// since this isn't a runtime error, but a configuration error.
	//
	executor := NewNodeExecutor(r.Encryptor, r.Registry, "http://localhost", "http://localhost")
	err := executor.LockAndProcessNodeExecution(execution.ID)
	require.NoError(t, err)

	//
	// Verify the execution was marked as failed with an error reason.
	//
	failedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, failedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultReasonError, failedExecution.ResultReason)
	assert.Contains(t, failedExecution.ResultMessage, "error building configuration for execution of node")
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
