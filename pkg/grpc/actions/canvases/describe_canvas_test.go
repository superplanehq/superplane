package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__DescribeCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), "invalid-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("returns canvas with metadata/spec/status structure", func(t *testing.T) {
		//
		// Create a canvas with nodes and edges
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "First Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "node-2",
					Name:   "Second Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{
				{
					SourceID: "node-1",
					TargetID: "node-2",
					Channel:  "default",
				},
			},
		)

		require.NoError(t, database.Conn().
			Model(&models.WorkflowNode{}).
			Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").
			Update("state", models.WorkflowNodeStatePaused).
			Error)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)

		//
		// Verify metadata structure
		//
		require.NotNil(t, response.Canvas.Metadata)
		assert.Equal(t, workflow.ID.String(), response.Canvas.Metadata.Id)
		assert.Equal(t, workflow.OrganizationID.String(), response.Canvas.Metadata.OrganizationId)
		assert.Equal(t, workflow.Name, response.Canvas.Metadata.Name)
		assert.Equal(t, workflow.Description, response.Canvas.Metadata.Description)
		assert.NotNil(t, response.Canvas.Metadata.CreatedAt)
		assert.NotNil(t, response.Canvas.Metadata.UpdatedAt)
		assert.NotNil(t, response.Canvas.Metadata.CreatedBy)

		//
		// Verify spec structure
		//
		require.NotNil(t, response.Canvas.Spec)
		assert.Len(t, response.Canvas.Spec.Nodes, 2)
		assert.Equal(t, "node-1", response.Canvas.Spec.Nodes[0].Id)
		assert.Equal(t, "First Node", response.Canvas.Spec.Nodes[0].Name)
		assert.Equal(t, "node-2", response.Canvas.Spec.Nodes[1].Id)
		assert.Equal(t, "Second Node", response.Canvas.Spec.Nodes[1].Name)

		var pausedNode *compb.Node
		for _, node := range response.Canvas.Spec.Nodes {
			if node.Id == "node-1" {
				pausedNode = node
				break
			}
		}
		require.NotNil(t, pausedNode)
		assert.True(t, pausedNode.Paused)

		assert.Len(t, response.Canvas.Spec.Edges, 1)
		assert.Equal(t, "node-1", response.Canvas.Spec.Edges[0].SourceId)
		assert.Equal(t, "node-2", response.Canvas.Spec.Edges[0].TargetId)
		assert.Equal(t, "default", response.Canvas.Spec.Edges[0].Channel)

		//
		// Verify status structure exists (even if empty)
		//
		require.NotNil(t, response.Canvas.Status)
		assert.NotNil(t, response.Canvas.Status.LastExecutions)
		assert.NotNil(t, response.Canvas.Status.NextQueueItems)
		assert.NotNil(t, response.Canvas.Status.LastEvents)
	})

	t.Run("status includes last execution per node", func(t *testing.T) {
		//
		// Create a canvas with two nodes
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "First Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "node-2",
					Name:   "Second Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events for executions
		//
		rootEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		rootEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)
		event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)

		//
		// Create multiple executions for node-1 (older one first)
		//
		oldExecution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent1.ID, event1.ID, nil)
		// Wait a bit to ensure different timestamps
		support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent1.ID, event1.ID, nil)

		//
		// Create one execution for node-2
		//
		support.CreateWorkflowNodeExecution(t, workflow.ID, "node-2", rootEvent2.ID, event2.ID, nil)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify we get exactly one execution per node (the latest one)
		//
		assert.Len(t, response.Canvas.Status.LastExecutions, 2)

		// Verify the latest execution for node-1 is NOT the old one
		var node1Execution *models.WorkflowNodeExecution
		var node2Execution *models.WorkflowNodeExecution
		for _, exec := range response.Canvas.Status.LastExecutions {
			if exec.NodeId == "node-1" {
				node1Execution = &models.WorkflowNodeExecution{ID: uuid.MustParse(exec.Id)}
			}
			if exec.NodeId == "node-2" {
				node2Execution = &models.WorkflowNodeExecution{ID: uuid.MustParse(exec.Id)}
			}
		}

		require.NotNil(t, node1Execution)
		require.NotNil(t, node2Execution)
		assert.NotEqual(t, oldExecution.ID.String(), node1Execution.ID.String(), "Should return the latest execution, not the old one")
	})

	t.Run("status includes next queue item per node", func(t *testing.T) {
		//
		// Create a canvas with two nodes
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "First Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "node-2",
					Name:   "Second Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events for queue items
		//
		rootEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		rootEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)
		event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)

		//
		// Create multiple queue items for node-1 (oldest one first)
		//
		support.CreateWorkflowQueueItem(t, workflow.ID, "node-1", rootEvent1.ID, event1.ID)
		// Wait a bit to ensure different timestamps
		support.CreateWorkflowQueueItem(t, workflow.ID, "node-1", rootEvent1.ID, event1.ID)

		//
		// Create one queue item for node-2
		//
		support.CreateWorkflowQueueItem(t, workflow.ID, "node-2", rootEvent2.ID, event2.ID)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify we get exactly one queue item per node (the oldest/next one)
		//
		assert.Len(t, response.Canvas.Status.NextQueueItems, 2)

		// Verify each node has a queue item
		var node1QueueItem *models.WorkflowNodeQueueItem
		var node2QueueItem *models.WorkflowNodeQueueItem
		for _, item := range response.Canvas.Status.NextQueueItems {
			if item.NodeId == "node-1" {
				node1QueueItem = &models.WorkflowNodeQueueItem{ID: uuid.MustParse(item.Id)}
			}
			if item.NodeId == "node-2" {
				node2QueueItem = &models.WorkflowNodeQueueItem{ID: uuid.MustParse(item.Id)}
			}
		}

		require.NotNil(t, node1QueueItem)
		require.NotNil(t, node2QueueItem)
	})

	t.Run("status includes last event per node", func(t *testing.T) {
		//
		// Create a canvas with two nodes
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "First Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "node-2",
					Name:   "Second Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create multiple events for node-1 (older one first)
		//
		oldEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		// Wait a bit to ensure different timestamps
		support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

		//
		// Create one event for node-2
		//
		support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify we get exactly one event per node (the latest one)
		//
		assert.Len(t, response.Canvas.Status.LastEvents, 2)

		// Verify the latest event for node-1 is NOT the old one
		var node1Event *models.WorkflowEvent
		var node2Event *models.WorkflowEvent
		for _, event := range response.Canvas.Status.LastEvents {
			if event.NodeId == "node-1" {
				node1Event = &models.WorkflowEvent{ID: uuid.MustParse(event.Id)}
			}
			if event.NodeId == "node-2" {
				node2Event = &models.WorkflowEvent{ID: uuid.MustParse(event.Id)}
			}
		}

		require.NotNil(t, node1Event)
		require.NotNil(t, node2Event)
		assert.NotEqual(t, oldEvent.ID.String(), node1Event.ID.String(), "Should return the latest event, not the old one")
	})

	t.Run("status is empty when no executions or queue items exist", func(t *testing.T) {
		//
		// Create a canvas with nodes but no executions or queue items
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "First Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify status exists but is empty
		//
		assert.Empty(t, response.Canvas.Status.LastExecutions)
		assert.Empty(t, response.Canvas.Status.NextQueueItems)
		assert.Empty(t, response.Canvas.Status.LastEvents)
	})

	t.Run("status excludes executions for deleted nodes", func(t *testing.T) {
		//
		// Create a canvas with three nodes
		//
		workflow, workflowNodes := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "active-node-1",
					Name:   "Active Node 1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "active-node-2",
					Name:   "Active Node 2",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "deleted-node",
					Name:   "Deleted Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events for executions
		//
		rootEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-1", "default", nil)
		event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-1", "default", nil)
		rootEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-2", "default", nil)
		event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-2", "default", nil)
		rootEvent3 := support.EmitWorkflowEventForNode(t, workflow.ID, "deleted-node", "default", nil)
		event3 := support.EmitWorkflowEventForNode(t, workflow.ID, "deleted-node", "default", nil)

		//
		// Create executions for all nodes
		//
		activeExec1 := support.CreateWorkflowNodeExecution(t, workflow.ID, "active-node-1", rootEvent1.ID, event1.ID, nil)
		activeExec2 := support.CreateWorkflowNodeExecution(t, workflow.ID, "active-node-2", rootEvent2.ID, event2.ID, nil)
		deletedExec := support.CreateWorkflowNodeExecution(t, workflow.ID, "deleted-node", rootEvent3.ID, event3.ID, nil)

		//
		// Delete one node (soft delete)
		//
		var deletedNode *models.WorkflowNode
		for i := range workflowNodes {
			if workflowNodes[i].NodeID == "deleted-node" {
				deletedNode = &workflowNodes[i]
				break
			}
		}
		require.NotNil(t, deletedNode)
		err := database.Conn().Delete(deletedNode).Error
		require.NoError(t, err)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify we only get executions for active nodes, not deleted ones
		//
		assert.Len(t, response.Canvas.Status.LastExecutions, 2)

		// Verify active node executions are included
		executionIDs := make(map[string]bool)
		for _, exec := range response.Canvas.Status.LastExecutions {
			executionIDs[exec.Id] = true
			// Verify it's not the deleted node's execution
			assert.NotEqual(t, "deleted-node", exec.NodeId)
		}

		assert.True(t, executionIDs[activeExec1.ID.String()])
		assert.True(t, executionIDs[activeExec2.ID.String()])
		assert.False(t, executionIDs[deletedExec.ID.String()])
	})

	t.Run("status excludes queue items for deleted nodes", func(t *testing.T) {
		//
		// Create a canvas with three nodes
		//
		workflow, workflowNodes := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "active-node-1",
					Name:   "Active Node 1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "active-node-2",
					Name:   "Active Node 2",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "deleted-node",
					Name:   "Deleted Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events for queue items
		//
		rootEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-1", "default", nil)
		event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-1", "default", nil)
		rootEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-2", "default", nil)
		event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-2", "default", nil)
		rootEvent3 := support.EmitWorkflowEventForNode(t, workflow.ID, "deleted-node", "default", nil)
		event3 := support.EmitWorkflowEventForNode(t, workflow.ID, "deleted-node", "default", nil)

		//
		// Create queue items for all nodes
		//
		activeQI1 := support.CreateWorkflowQueueItem(t, workflow.ID, "active-node-1", rootEvent1.ID, event1.ID)
		activeQI2 := support.CreateWorkflowQueueItem(t, workflow.ID, "active-node-2", rootEvent2.ID, event2.ID)
		deletedQI := support.CreateWorkflowQueueItem(t, workflow.ID, "deleted-node", rootEvent3.ID, event3.ID)

		//
		// Delete one node (soft delete)
		//
		var deletedNode *models.WorkflowNode
		for i := range workflowNodes {
			if workflowNodes[i].NodeID == "deleted-node" {
				deletedNode = &workflowNodes[i]
				break
			}
		}
		require.NotNil(t, deletedNode)
		err := database.Conn().Delete(deletedNode).Error
		require.NoError(t, err)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify we only get queue items for active nodes, not deleted ones
		//
		assert.Len(t, response.Canvas.Status.NextQueueItems, 2)

		// Verify active node queue items are included
		queueItemIDs := make(map[string]bool)
		for _, item := range response.Canvas.Status.NextQueueItems {
			queueItemIDs[item.Id] = true
			// Verify it's not the deleted node's queue item
			assert.NotEqual(t, "deleted-node", item.NodeId)
		}

		assert.True(t, queueItemIDs[activeQI1.ID.String()])
		assert.True(t, queueItemIDs[activeQI2.ID.String()])
		assert.False(t, queueItemIDs[deletedQI.ID.String()])
	})

	t.Run("status excludes events for deleted nodes", func(t *testing.T) {
		//
		// Create a canvas with three nodes
		//
		workflow, workflowNodes := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "active-node-1",
					Name:   "Active Node 1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "active-node-2",
					Name:   "Active Node 2",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "deleted-node",
					Name:   "Deleted Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events for all nodes
		//
		activeEvent1 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-1", "default", nil)
		activeEvent2 := support.EmitWorkflowEventForNode(t, workflow.ID, "active-node-2", "default", nil)
		deletedEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "deleted-node", "default", nil)

		//
		// Delete one node (soft delete)
		//
		var deletedNode *models.WorkflowNode
		for i := range workflowNodes {
			if workflowNodes[i].NodeID == "deleted-node" {
				deletedNode = &workflowNodes[i]
				break
			}
		}
		require.NotNil(t, deletedNode)
		err := database.Conn().Delete(deletedNode).Error
		require.NoError(t, err)

		//
		// Describe the canvas
		//
		response, err := DescribeCanvas(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Canvas.Status)

		//
		// Verify we only get events for active nodes, not deleted ones
		//
		assert.Len(t, response.Canvas.Status.LastEvents, 2)

		// Verify active node events are included
		eventIDs := make(map[string]bool)
		for _, event := range response.Canvas.Status.LastEvents {
			eventIDs[event.Id] = true
			// Verify it's not the deleted node's event
			assert.NotEqual(t, "deleted-node", event.NodeId)
		}

		assert.True(t, eventIDs[activeEvent1.ID.String()])
		assert.True(t, eventIDs[activeEvent2.ID.String()])
		assert.False(t, eventIDs[deletedEvent.ID.String()])
	})
}
