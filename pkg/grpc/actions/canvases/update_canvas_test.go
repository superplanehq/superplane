package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func Test__UpdateCanvas__NodeRemovalUseSoftDelete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-2", event.ID, event.ID, nil)

	require.NoError(t, models.CreateNodeExecutionKVInTransaction(
		database.Conn(),
		canvas.ID,
		"node-2",
		execution.ID,
		"test-key",
		"test-value",
	))

	canvasPb := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "node-1",
					Name: "Node 1",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		canvasPb,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed when removing nodes with execution KVs")

	var normalCount int64
	database.Conn().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").Count(&normalCount)
	assert.Equal(t, int64(0), normalCount, "node-2 should not be visible in normal queries (soft deleted)")

	var unscopedCount int64
	database.Conn().Unscoped().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").Count(&unscopedCount)
	assert.Equal(t, int64(1), unscopedCount, "node-2 should be visible with Unscoped() (soft deleted, not hard deleted)")

	var softDeletedNode models.CanvasNode
	err = database.Conn().Unscoped().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").First(&softDeletedNode).Error
	require.NoError(t, err, "should be able to find soft deleted node with Unscoped()")
	assert.True(t, softDeletedNode.DeletedAt.Valid, "node-2 should have valid deleted_at timestamp")

	var activeCount int64
	database.Conn().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Count(&activeCount)
	assert.Equal(t, int64(1), activeCount, "node-1 should still be active")

	var activeNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").First(&activeNode).Error
	require.NoError(t, err, "should be able to find active node")
	assert.False(t, activeNode.DeletedAt.Valid, "node-1 should not have deleted_at timestamp")

	var kvCount int64
	database.Conn().Model(&models.CanvasNodeExecutionKV{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").Count(&kvCount)
	assert.Equal(t, int64(1), kvCount, "execution KV should still exist (FK constraint satisfied by soft deleted node)")

	var executionCount int64
	database.Conn().Model(&models.CanvasNodeExecution{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").Count(&executionCount)
	assert.Equal(t, int64(1), executionCount, "execution should still exist (FK constraint satisfied by soft deleted node)")
}

func Test__UpdateCanvas__RemapConflictingNodeIDs(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
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

	removeNodePB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "node-2",
					Name: "Node 2",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		removeNodePB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed when removing nodes")

	remapCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "node-1",
					Name: "Node 1",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "node-2",
					Name: "Node 2",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{
				{
					SourceId: "node-1",
					TargetId: "node-2",
					Channel:  "default",
				},
			},
		},
	}

	response, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		remapCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed when remapping conflicting node IDs")
	require.NotNil(t, response.Canvas)
	require.NotNil(t, response.Canvas.Spec)

	var remappedNode *componentpb.NodeDefinition
	for _, node := range response.Canvas.Spec.Nodes {
		if node.GetName() == "Node 1" {
			remappedNode = node
			break
		}
	}
	require.NotNil(t, remappedNode, "expected remapped node to exist")
	assert.NotEqual(t, "node-1", remappedNode.GetId(), "remapped node should not keep the soft-deleted ID")

	var remappedEdge *componentpb.Edge
	for _, edge := range response.Canvas.Spec.Edges {
		if edge.GetTargetId() == "node-2" {
			remappedEdge = edge
			break
		}
	}
	require.NotNil(t, remappedEdge, "expected remapped edge to exist")
	assert.Equal(t, remappedNode.GetId(), remappedEdge.GetSourceId(), "edge should point at remapped node ID")

	var activeCount int64
	database.Conn().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Count(&activeCount)
	assert.Equal(t, int64(0), activeCount, "soft-deleted node should not be active")

	var unscopedCount int64
	database.Conn().Unscoped().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").Count(&unscopedCount)
	assert.Equal(t, int64(1), unscopedCount, "soft-deleted node should remain in history")

	var remappedCount int64
	database.Conn().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, remappedNode.GetId()).Count(&remappedCount)
	assert.Equal(t, int64(1), remappedCount, "remapped node should be active")
}

func Test__UpdateCanvas__ErroredNodesCanExist(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	errorReason := "Simulated setup error during component initialization"
	err := database.Conn().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").
		Updates(map[string]interface{}{
			"state":        models.CanvasNodeStateError,
			"state_reason": errorReason,
		}).Error
	require.NoError(t, err, "should be able to set node to error state")

	updatedCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name + " Updated",
			Description: canvas.Description + " Updated",
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "node-1",
					Name: "Node 1 Updated",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "node-2",
					Name: "Node 2 Updated",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		updatedCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed even with existing errored nodes")

	var goodNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").First(&goodNode).Error
	require.NoError(t, err, "should be able to find good node")
	assert.Equal(t, models.CanvasNodeStateReady, goodNode.State, "good node should be ready")
	assert.Nil(t, goodNode.StateReason, "good node should not have state reason")
	assert.Equal(t, "Node 1 Updated", goodNode.Name, "good node name should be updated")

	var previouslyErroredNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-2").First(&previouslyErroredNode).Error
	require.NoError(t, err, "should be able to find previously errored node")
	assert.Equal(t, models.CanvasNodeStateReady, previouslyErroredNode.State, "previously errored node should be reset to ready")
	assert.Nil(t, previouslyErroredNode.StateReason, "previously errored node should have cleared state reason")
	assert.Equal(t, "Node 2 Updated", previouslyErroredNode.Name, "previously errored node name should be updated")
}

func Test__UpdateCanvas__ErroredNodeResetOnUpdate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				State:  models.CanvasNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	errorReason := "Previous error"
	err := database.Conn().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
		Updates(map[string]interface{}{
			"state":        models.CanvasNodeStateError,
			"state_reason": errorReason,
		}).Error
	require.NoError(t, err, "should be able to manually set node to error state")

	var initialNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").First(&initialNode).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateError, initialNode.State)
	assert.NotNil(t, initialNode.StateReason)
	assert.Equal(t, "Previous error", *initialNode.StateReason)

	updatedCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "node-1",
					Name: "Node 1 Updated",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		updatedCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed and reset errored node")

	var updatedNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").First(&updatedNode).Error
	require.NoError(t, err, "should be able to find updated node")
	assert.Equal(t, models.CanvasNodeStateReady, updatedNode.State, "previously errored node should now be ready")
	assert.Nil(t, updatedNode.StateReason, "error reason should be cleared")
	assert.Equal(t, "Node 1 Updated", updatedNode.Name, "node name should be updated")
}

func Test__UpdateCanvas__NonErroredNodesKeepState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "ready-node",
				Name:   "Ready Node",
				Type:   models.NodeTypeComponent,
				State:  models.CanvasNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "processing-node",
				Name:   "Processing Node",
				Type:   models.NodeTypeComponent,
				State:  models.CanvasNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "errored-node",
				Name:   "Errored Node",
				Type:   models.NodeTypeComponent,
				State:  models.CanvasNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	err := database.Conn().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "processing-node").
		Update("state", models.CanvasNodeStateProcessing).Error
	require.NoError(t, err)

	errorReason := "Previous error"
	err = database.Conn().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, "errored-node").
		Updates(map[string]interface{}{
			"state":        models.CanvasNodeStateError,
			"state_reason": errorReason,
		}).Error
	require.NoError(t, err)

	updatedCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "ready-node",
					Name: "Ready Node Updated",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "processing-node",
					Name: "Processing Node Updated",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "errored-node",
					Name: "Errored Node Updated",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		updatedCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed")

	readyNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, "ready-node")
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, readyNode.State, "ready node should stay ready")
	assert.Nil(t, readyNode.StateReason, "ready node should not have error reason")
	assert.Equal(t, "Ready Node Updated", readyNode.Name, "ready node name should be updated")

	processingNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, "processing-node")
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateProcessing, processingNode.State, "processing node should stay processing")
	assert.Nil(t, processingNode.StateReason, "processing node should not have error reason")
	assert.Equal(t, "Processing Node Updated", processingNode.Name, "processing node name should be updated")

	erroredNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, "errored-node")
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeStateReady, erroredNode.State, "errored node should be reset to ready")
	assert.Nil(t, erroredNode.StateReason, "errored node error reason should be cleared")
	assert.Equal(t, "Errored Node Updated", erroredNode.Name, "errored node name should be updated")
}

func Test__UpdateCanvas__ValidationErrorsPersisted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	updatedCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "valid-node",
					Name: "Valid Node",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "invalid-node",
					Name: "Invalid Node",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "nonexistent-component",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		updatedCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed even with validation errors")

	validNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, "valid-node")
	require.NoError(t, err, "should be able to find valid node")
	assert.Equal(t, models.CanvasNodeStateReady, validNode.State, "valid node should be ready")
	assert.Nil(t, validNode.StateReason, "valid node should not have error reason")

	invalidNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, "invalid-node")
	require.NoError(t, err, "should be able to find invalid node")
	assert.Equal(t, models.CanvasNodeStateError, invalidNode.State, "invalid node should be in error state")
	assert.NotNil(t, invalidNode.StateReason, "invalid node should have error reason")
	assert.Contains(t, *invalidNode.StateReason, "nonexistent-component", "error reason should mention the invalid component")
}

func Test__UpdateCanvas__ErroredNodeBecomesValidAgain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	invalidCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "test-node",
					Name: "Test Node",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "nonexistent-component",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		invalidCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed even with validation errors")

	var testNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "test-node").First(&testNode).Error
	require.NoError(t, err, "should be able to find test node")
	assert.Equal(t, models.CanvasNodeStateError, testNode.State, "node should be in error state")
	assert.NotNil(t, testNode.StateReason, "node should have error reason")

	validCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "test-node",
					Name: "Test Node",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		validCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed with valid configuration")

	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "test-node").First(&testNode).Error
	require.NoError(t, err, "should be able to find test node")
	assert.Equal(t, models.CanvasNodeStateReady, testNode.State, "node should now be in ready state")
	assert.Nil(t, testNode.StateReason, "node should not have error reason")
}

func Test__UpdateCanvas__WidgetNodesHandled(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	annotationText := "This is an annotation describing the workflow"
	annotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": annotationText,
	})

	initialCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:            "annotation-1",
					Name:          "Workflow Note",
					Type:          componentpb.NodeDefinition_TYPE_WIDGET,
					Configuration: annotationConfig,
					Widget: &componentpb.NodeDefinition_WidgetRef{
						Name: "annotation",
					},
				},
				{
					Id:   "component-1",
					Name: "Regular Component",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	updatedCanvas, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		initialCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed with widget nodes")

	var widgetNodeCount int64
	database.Conn().Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "annotation-1").Count(&widgetNodeCount)
	assert.Equal(t, int64(0), widgetNodeCount, "widget nodes should not be persisted in workflow_nodes table")

	var componentNode models.CanvasNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", canvas.ID, "component-1").First(&componentNode).Error
	require.NoError(t, err, "should be able to find component node")
	assert.Equal(t, models.NodeTypeComponent, componentNode.Type, "component node should have correct type")

	assert.NotNil(t, updatedCanvas.Canvas.Spec.Nodes, "workflow should have nodes in spec")
	var foundWidget *componentpb.NodeDefinition
	for _, node := range updatedCanvas.Canvas.Spec.Nodes {
		if node.Id == "annotation-1" {
			foundWidget = node
			break
		}
	}
	require.NotNil(t, foundWidget, "should find widget in workflow nodes JSON")
	assert.Equal(t, componentpb.NodeDefinition_TYPE_WIDGET, foundWidget.Type, "widget should have correct type")
	assert.Equal(t, "annotation", foundWidget.Widget.Name, "widget should have correct name")
	assert.Equal(t, annotationText, foundWidget.Configuration.AsMap()["text"], "widget text should match")

	updatedAnnotationText := "This is an updated annotation"
	updatedAnnotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": updatedAnnotationText,
	})

	updatedCanvasPB := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:            "annotation-1",
					Name:          "Workflow Note Updated",
					Type:          componentpb.NodeDefinition_TYPE_WIDGET,
					Configuration: updatedAnnotationConfig,
					Widget: &componentpb.NodeDefinition_WidgetRef{
						Name: "annotation",
					},
				},
				{
					Id:   "component-1",
					Name: "Regular Component",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	finalUpdatedCanvas, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		updatedCanvasPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateCanvas should succeed when updating widget nodes")

	// Verify updated widget is in JSON
	var updatedWidget *componentpb.NodeDefinition
	for _, node := range finalUpdatedCanvas.Canvas.Spec.Nodes {
		if node.Id == "annotation-1" {
			updatedWidget = node
			break
		}
	}
	require.NotNil(t, updatedWidget, "should find updated widget in workflow nodes JSON")
	assert.Equal(t, "Workflow Note Updated", updatedWidget.Name, "widget name should be updated")
	assert.Equal(t, updatedAnnotationText, updatedWidget.Configuration.AsMap()["text"], "widget text should be updated")
}

func Test__UpdateCanvas__WidgetNodesCannotConnect(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	annotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": "This is an annotation",
	})

	workflowWithWidgetAsSource := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:            "annotation-1",
					Name:          "Annotation Note",
					Type:          componentpb.NodeDefinition_TYPE_WIDGET,
					Configuration: annotationConfig,
					Widget: &componentpb.NodeDefinition_WidgetRef{
						Name: "annotation",
					},
				},
				{
					Id:   "component-1",
					Name: "Component",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{
				{
					SourceId: "annotation-1",
					TargetId: "component-1",
					Channel:  "default",
				},
			},
		},
	}

	_, err := UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		workflowWithWidgetAsSource,
		"http://localhost:3000/api/v1",
	)
	require.Error(t, err, "UpdateWorkflow should fail when widget node is used as source")
	assert.Contains(t, err.Error(), "widget nodes cannot be used as source nodes")

	workflowWithWidgetAsTarget := &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Name:        canvas.Name,
			Description: canvas.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: []*componentpb.NodeDefinition{
				{
					Id:   "component-1",
					Name: "Component",
					Type: componentpb.NodeDefinition_TYPE_COMPONENT,
					Component: &componentpb.NodeDefinition_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:            "annotation-1",
					Name:          "Annotation Note",
					Type:          componentpb.NodeDefinition_TYPE_WIDGET,
					Configuration: annotationConfig,
					Widget: &componentpb.NodeDefinition_WidgetRef{
						Name: "annotation",
					},
				},
			},
			Edges: []*componentpb.Edge{
				{
					SourceId: "component-1",
					TargetId: "annotation-1",
					Channel:  "default",
				},
			},
		},
	}

	_, err = UpdateCanvas(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		canvas.ID.String(),
		workflowWithWidgetAsTarget,
		"http://localhost:3000/api/v1",
	)
	require.Error(t, err, "UpdateCanvas should fail when widget node is used as target")
	assert.Contains(t, err.Error(), "widget nodes cannot be used as target nodes")
}
