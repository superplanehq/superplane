package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

func Test__WorkflowEventRouter_ProcessRootEvent(t *testing.T) {
	router := NewWorkflowEventRouter()
	r := support.Setup(t)

	//
	// Create a simple workflow with just a trigger and a component nodes.
	//
	node1 := "trigger-1"
	node2 := "component-1"
	workflow, _ := createTestWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{NodeID: node1, Type: models.NodeTypeTrigger},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	//
	// Create the root event for the trigger node, and process it.
	//
	event := emitTestEventForNode(t, workflow.ID, node1, "default", nil)
	err := router.LockAndProcessEvent(*event)
	require.NoError(t, err)

	//
	// Verify event was marked as routed and
	// new queue item was created for the target node.
	//
	updatedEvent, err := models.FindWorkflowEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowEventStateRouted, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(workflow.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, node2, queueItems[0].NodeID)
	assert.Equal(t, event.ID, queueItems[0].EventID)
}

func Test__WorkflowEventRouter_ProcessExecutionEvent(t *testing.T) {
	router := NewWorkflowEventRouter()
	r := support.Setup(t)

	trigger1 := "trigger-1"
	node1 := "component-1"
	node2 := "component-2"
	workflow, _ := createTestWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{NodeID: trigger1, Type: models.NodeTypeTrigger},
			{NodeID: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger1, TargetID: node1, Channel: "default"},
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	//
	// Create root event for trigger node,
	// and create execution with output event for node1.
	//
	triggerEvent := emitTestEventForNode(t, workflow.ID, trigger1, "default", nil)
	execution := createTestExecution(t, workflow.ID, node1, triggerEvent.ID, triggerEvent.ID, nil)
	outputEvent := emitTestEventForNode(t, workflow.ID, node1, "default", &execution.ID)

	//
	// Process node1 output event and verify it was routed properly.
	//
	err := router.LockAndProcessEvent(*outputEvent)
	require.NoError(t, err)

	updatedEvent, err := models.FindWorkflowEvent(outputEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowEventStateRouted, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(workflow.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, node2, queueItems[0].NodeID)
	assert.Equal(t, outputEvent.ID, queueItems[0].EventID)
}

func Test__WorkflowEventRouter_CustomComponent_RespectsOutputChannels(t *testing.T) {
	router := NewWorkflowEventRouter()
	r := support.Setup(t)

	//
	// Create a blueprint with this structure:
	//
	//   if-1 --true--> up
	//        --false-> down
	//
	blueprint := createTestBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{ID: "if-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
		[]models.BlueprintOutputChannel{
			{Name: "up", NodeID: "if-1", NodeOutputChannel: "true"},
			{Name: "down", NodeID: "if-1", NodeOutputChannel: "false"},
		},
	)

	// Create a workflow that uses this custom component
	customComponentNode := "blueprint-1"
	workflow, _ := createTestWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{
				NodeID: "trigger-1",
				Type:   models.NodeTypeTrigger,
			},
			{
				NodeID: customComponentNode,
				Type:   models.NodeTypeBlueprint,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()},
				}),
			},
			{
				NodeID: "next-up",
				Type:   models.NodeTypeComponent,
			},
			{
				NodeID: "next-down",
				Type:   models.NodeTypeComponent,
			},
		},
		[]models.Edge{
			{SourceID: "trigger-1", TargetID: customComponentNode, Channel: "default"},
			{SourceID: customComponentNode, TargetID: "next-up", Channel: "up"},
			{SourceID: customComponentNode, TargetID: "next-down", Channel: "down"},
		},
	)

	//
	// Create parent execution for the custom component
	//
	rootEvent := emitTestEventForNode(t, workflow.ID, "trigger-1", "default", nil)
	require.NoError(t, rootEvent.Routed())
	parentExecution := createTestExecution(t, workflow.ID, customComponentNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Create and pass child execution,
	// emit output event on "true" channel.
	//
	childExecution := createTestExecution(t, workflow.ID, customComponentNode+":if-1", rootEvent.ID, rootEvent.ID, &parentExecution.ID)
	require.NoError(t, childExecution.Pass(map[string][]any{
		"true": {map[string]any{}},
	}))

	//
	// Process the child output event,
	// verify parent execution is completed,
	// and verify parent execution emitted events only on the "up" channel, NOT on "down"
	//
	events, err := models.ListWorkflowEvents(workflow.ID, customComponentNode+":if-1", 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	childOutputEvent := events[0]
	err = router.LockAndProcessEvent(childOutputEvent)
	require.NoError(t, err)

	parent, err := models.FindNodeExecution(workflow.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, parent.State)

	parentOutputEvents, err := models.ListWorkflowEvents(workflow.ID, customComponentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, filterEventsByChannel(parentOutputEvents, "up"), 1)
	assert.Len(t, filterEventsByChannel(parentOutputEvents, "down"), 0)
}

func TestWorkflowEventRouter__CustomComponent_MultipleOutputs(t *testing.T) {
	router := NewWorkflowEventRouter()
	r := support.Setup(t)

	//
	// Create a blueprint with this structure:
	//
	//   filter-1 --default--> default
	//
	blueprint := createTestBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{ID: "filter-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
		[]models.BlueprintOutputChannel{
			{Name: "default", NodeID: "filter-1", NodeOutputChannel: "default"},
		},
	)

	// Create a workflow that uses this custom component
	customComponentNode := "blueprint-1"
	workflow, _ := createTestWorkflow(
		t,
		r.Organization.ID,
		[]models.WorkflowNode{
			{
				NodeID: "trigger-1",
				Type:   models.NodeTypeTrigger,
			},
			{
				NodeID: customComponentNode,
				Type:   models.NodeTypeBlueprint,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()},
				}),
			},
			{
				NodeID: "next",
				Type:   models.NodeTypeComponent,
			},
		},
		[]models.Edge{
			{SourceID: "trigger-1", TargetID: customComponentNode, Channel: "default"},
			{SourceID: customComponentNode, TargetID: "next", Channel: "default"},
		},
	)

	//
	// Create parent execution for the custom component
	//
	rootEvent := emitTestEventForNode(t, workflow.ID, "trigger-1", "default", nil)
	require.NoError(t, rootEvent.Routed())
	parentExecution := createTestExecution(t, workflow.ID, customComponentNode, rootEvent.ID, rootEvent.ID, nil)

	//
	// Create and pass child execution, emitting 5 events.
	//
	childExecution := createTestExecution(t, workflow.ID, customComponentNode+":filter-1", rootEvent.ID, rootEvent.ID, &parentExecution.ID)
	require.NoError(t, childExecution.Pass(map[string][]any{
		"default": {
			map[string]any{},
			map[string]any{},
			map[string]any{},
			map[string]any{},
			map[string]any{},
		},
	}))

	//
	// Process one of the child output events,
	// verify parent execution is completed,
	// emitting 5 events too.
	//
	events, err := models.ListWorkflowEvents(workflow.ID, customComponentNode+":filter-1", 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 5)
	childOutputEvent := events[0]
	err = router.LockAndProcessEvent(childOutputEvent)
	require.NoError(t, err)

	parent, err := models.FindNodeExecution(workflow.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeExecutionStateFinished, parent.State)

	parentOutputEvents, err := models.ListWorkflowEvents(workflow.ID, customComponentNode, 10, nil)
	require.NoError(t, err)
	assert.Len(t, filterEventsByChannel(parentOutputEvents, "default"), 5)
}

//
// Helper functions
//

func createTestWorkflow(t *testing.T, orgID uuid.UUID, nodes []models.WorkflowNode, edges []models.Edge) (*models.Workflow, []models.WorkflowNode) {
	now := time.Now()

	//
	// Create workflow
	//
	workflow := &models.Workflow{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           support.RandomName("workflow"),
		Description:    "Test workflow",
		Edges:          datatypes.NewJSONSlice(edges),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Create(workflow).Error)

	//
	// Create workflow nodes
	//
	for _, node := range nodes {
		node.WorkflowID = workflow.ID
		node.State = models.WorkflowNodeStateReady
		node.CreatedAt = &now
		node.UpdatedAt = &now
		require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&node).Error)
	}

	return workflow, nodes
}

func createTestBlueprint(t *testing.T, orgID uuid.UUID, nodes []models.Node, edges []models.Edge, outputChannels []models.BlueprintOutputChannel) *models.Blueprint {
	now := time.Now()

	blueprint := models.Blueprint{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           support.RandomName("blueprint"),
		Nodes:          datatypes.NewJSONSlice(nodes),
		Edges:          datatypes.NewJSONSlice(edges),
		OutputChannels: datatypes.NewJSONSlice(outputChannels),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Create(&blueprint).Error)

	return &blueprint
}

func createTestExecution(t *testing.T, workflowID uuid.UUID, nodeID string, rootEventID uuid.UUID, eventID uuid.UUID, parentExecutionID *uuid.UUID) *models.WorkflowNodeExecution {
	now := time.Now()
	execution := models.WorkflowNodeExecution{
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		RootEventID:       rootEventID,
		EventID:           eventID,
		ParentExecutionID: parentExecutionID,
		State:             models.WorkflowNodeExecutionStatePending,
		Configuration:     datatypes.NewJSONType(map[string]any{}),
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}

	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func filterEventsByChannel(events []models.WorkflowEvent, channel string) []models.WorkflowEvent {
	var filtered []models.WorkflowEvent
	for _, event := range events {
		if event.Channel == channel {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func emitTestEventForNode(t *testing.T, workflowID uuid.UUID, nodeID string, channel string, executionID *uuid.UUID) *models.WorkflowEvent {
	now := time.Now()
	event := models.WorkflowEvent{
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		Channel:     channel,
		Data:        datatypes.NewJSONType[any](map[string]any{"key": "value"}),
		State:       models.WorkflowEventStatePending,
		ExecutionID: executionID,
		CreatedAt:   &now,
	}
	require.NoError(t, database.Conn().Clauses(clause.Returning{}).Create(&event).Error)
	return &event
}
