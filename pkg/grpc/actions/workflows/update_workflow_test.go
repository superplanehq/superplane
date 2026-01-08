package workflows

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func TestUpdateWorkflow_NodeRemovalUseSoftDelete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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

	event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)
	execution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-2", event.ID, event.ID, nil)

	require.NoError(t, models.CreateWorkflowNodeExecutionKVInTransaction(
		database.Conn(),
		workflow.ID,
		"node-2",
		execution.ID,
		"test-key",
		"test-value",
	))

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-1",
					Name: "Node 1",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed when removing nodes with execution KVs")

	var normalCount int64
	database.Conn().Model(&models.WorkflowNode{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").Count(&normalCount)
	assert.Equal(t, int64(0), normalCount, "node-2 should not be visible in normal queries (soft deleted)")

	var unscopedCount int64
	database.Conn().Unscoped().Model(&models.WorkflowNode{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").Count(&unscopedCount)
	assert.Equal(t, int64(1), unscopedCount, "node-2 should be visible with Unscoped() (soft deleted, not hard deleted)")

	var softDeletedNode models.WorkflowNode
	err = database.Conn().Unscoped().Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").First(&softDeletedNode).Error
	require.NoError(t, err, "should be able to find soft deleted node with Unscoped()")
	assert.True(t, softDeletedNode.DeletedAt.Valid, "node-2 should have valid deleted_at timestamp")

	var activeCount int64
	database.Conn().Model(&models.WorkflowNode{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").Count(&activeCount)
	assert.Equal(t, int64(1), activeCount, "node-1 should still be active")

	var activeNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").First(&activeNode).Error
	require.NoError(t, err, "should be able to find active node")
	assert.False(t, activeNode.DeletedAt.Valid, "node-1 should not have deleted_at timestamp")

	var kvCount int64
	database.Conn().Model(&models.WorkflowNodeExecutionKV{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").Count(&kvCount)
	assert.Equal(t, int64(1), kvCount, "execution KV should still exist (FK constraint satisfied by soft deleted node)")

	var executionCount int64
	database.Conn().Model(&models.WorkflowNodeExecution{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").Count(&executionCount)
	assert.Equal(t, int64(1), executionCount, "execution should still exist (FK constraint satisfied by soft deleted node)")
}

func TestUpdateWorkflow_ErroredNodesCanExist(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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
	err := database.Conn().Model(&models.WorkflowNode{}).
		Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").
		Updates(map[string]interface{}{
			"state":        models.WorkflowNodeStateError,
			"state_reason": errorReason,
		}).Error
	require.NoError(t, err, "should be able to set node to error state")

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name + " Updated",
			Description: workflow.Description + " Updated",
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-1",
					Name: "Node 1 Updated",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "node-2",
					Name: "Node 2 Updated",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed even with existing errored nodes")

	var goodNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").First(&goodNode).Error
	require.NoError(t, err, "should be able to find good node")
	assert.Equal(t, models.WorkflowNodeStateReady, goodNode.State, "good node should be ready")
	assert.Nil(t, goodNode.StateReason, "good node should not have state reason")
	assert.Equal(t, "Node 1 Updated", goodNode.Name, "good node name should be updated")

	var previouslyErroredNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-2").First(&previouslyErroredNode).Error
	require.NoError(t, err, "should be able to find previously errored node")
	assert.Equal(t, models.WorkflowNodeStateReady, previouslyErroredNode.State, "previously errored node should be reset to ready")
	assert.Nil(t, previouslyErroredNode.StateReason, "previously errored node should have cleared state reason")
	assert.Equal(t, "Node 2 Updated", previouslyErroredNode.Name, "previously errored node name should be updated")
}

func TestUpdateWorkflow_ErroredNodeResetOnUpdate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				State:  models.WorkflowNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	errorReason := "Previous error"
	err := database.Conn().Model(&models.WorkflowNode{}).
		Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").
		Updates(map[string]interface{}{
			"state":        models.WorkflowNodeStateError,
			"state_reason": errorReason,
		}).Error
	require.NoError(t, err, "should be able to manually set node to error state")

	var initialNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").First(&initialNode).Error
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateError, initialNode.State)
	assert.NotNil(t, initialNode.StateReason)
	assert.Equal(t, "Previous error", *initialNode.StateReason)

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "node-1",
					Name: "Node 1 Updated",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed and reset errored node")

	var updatedNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").First(&updatedNode).Error
	require.NoError(t, err, "should be able to find updated node")
	assert.Equal(t, models.WorkflowNodeStateReady, updatedNode.State, "previously errored node should now be ready")
	assert.Nil(t, updatedNode.StateReason, "error reason should be cleared")
	assert.Equal(t, "Node 1 Updated", updatedNode.Name, "node name should be updated")
}

func TestUpdateWorkflow_NonErroredNodesKeepState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "ready-node",
				Name:   "Ready Node",
				Type:   models.NodeTypeComponent,
				State:  models.WorkflowNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "processing-node",
				Name:   "Processing Node",
				Type:   models.NodeTypeComponent,
				State:  models.WorkflowNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "errored-node",
				Name:   "Errored Node",
				Type:   models.NodeTypeComponent,
				State:  models.WorkflowNodeStateReady,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	err := database.Conn().Model(&models.WorkflowNode{}).
		Where("workflow_id = ? AND node_id = ?", workflow.ID, "processing-node").
		Update("state", models.WorkflowNodeStateProcessing).Error
	require.NoError(t, err)

	errorReason := "Previous error"
	err = database.Conn().Model(&models.WorkflowNode{}).
		Where("workflow_id = ? AND node_id = ?", workflow.ID, "errored-node").
		Updates(map[string]interface{}{
			"state":        models.WorkflowNodeStateError,
			"state_reason": errorReason,
		}).Error
	require.NoError(t, err)

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "ready-node",
					Name: "Ready Node Updated",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "processing-node",
					Name: "Processing Node Updated",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "errored-node",
					Name: "Errored Node Updated",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed")

	var readyNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "ready-node").First(&readyNode).Error
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, readyNode.State, "ready node should stay ready")
	assert.Nil(t, readyNode.StateReason, "ready node should not have error reason")
	assert.Equal(t, "Ready Node Updated", readyNode.Name, "ready node name should be updated")

	var processingNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "processing-node").First(&processingNode).Error
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateProcessing, processingNode.State, "processing node should stay processing")
	assert.Nil(t, processingNode.StateReason, "processing node should not have error reason")
	assert.Equal(t, "Processing Node Updated", processingNode.Name, "processing node name should be updated")

	var erroredNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "errored-node").First(&erroredNode).Error
	require.NoError(t, err)
	assert.Equal(t, models.WorkflowNodeStateReady, erroredNode.State, "errored node should be reset to ready")
	assert.Nil(t, erroredNode.StateReason, "errored node error reason should be cleared")
	assert.Equal(t, "Errored Node Updated", erroredNode.Name, "errored node name should be updated")
}

func TestUpdateWorkflow_ValidationErrorsPersisted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "valid-node",
					Name: "Valid Node",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:   "invalid-node",
					Name: "Invalid Node",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "nonexistent-component",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed even with validation errors")

	var validNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "valid-node").First(&validNode).Error
	require.NoError(t, err, "should be able to find valid node")
	assert.Equal(t, models.WorkflowNodeStateReady, validNode.State, "valid node should be ready")
	assert.Nil(t, validNode.StateReason, "valid node should not have error reason")

	var invalidNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "invalid-node").First(&invalidNode).Error
	require.NoError(t, err, "should be able to find invalid node")
	assert.Equal(t, models.WorkflowNodeStateError, invalidNode.State, "invalid node should be in error state")
	assert.NotNil(t, invalidNode.StateReason, "invalid node should have error reason")
	assert.Contains(t, *invalidNode.StateReason, "nonexistent-component", "error reason should mention the invalid component")
}

func TestUpdateWorkflow_ErroredNodeBecomesValidAgain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	invalidWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "test-node",
					Name: "Test Node",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "nonexistent-component",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		invalidWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed even with validation errors")

	var testNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "test-node").First(&testNode).Error
	require.NoError(t, err, "should be able to find test node")
	assert.Equal(t, models.WorkflowNodeStateError, testNode.State, "node should be in error state")
	assert.NotNil(t, testNode.StateReason, "node should have error reason")

	validWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "test-node",
					Name: "Test Node",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	_, err = UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		validWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed with valid configuration")

	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "test-node").First(&testNode).Error
	require.NoError(t, err, "should be able to find test node")
	assert.Equal(t, models.WorkflowNodeStateReady, testNode.State, "node should now be in ready state")
	assert.Nil(t, testNode.StateReason, "node should not have error reason")
}

func TestUpdateWorkflow_WidgetNodesHandled(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	annotationText := "This is an annotation describing the workflow"
	annotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": annotationText,
	})

	initialWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:            "annotation-1",
					Name:          "Workflow Note",
					Type:          componentpb.Node_TYPE_WIDGET,
					Configuration: annotationConfig,
					Widget: &componentpb.Node_WidgetRef{
						Name: "annotation",
					},
				},
				{
					Id:   "component-1",
					Name: "Regular Component",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	updatedWorkflow, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		initialWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed with widget nodes")

	var widgetNodeCount int64
	database.Conn().Model(&models.WorkflowNode{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "annotation-1").Count(&widgetNodeCount)
	assert.Equal(t, int64(0), widgetNodeCount, "widget nodes should not be persisted in workflow_nodes table")

	var componentNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "component-1").First(&componentNode).Error
	require.NoError(t, err, "should be able to find component node")
	assert.Equal(t, models.NodeTypeComponent, componentNode.Type, "component node should have correct type")

	assert.NotNil(t, updatedWorkflow.Workflow.Spec.Nodes, "workflow should have nodes in spec")
	var foundWidget *componentpb.Node
	for _, node := range updatedWorkflow.Workflow.Spec.Nodes {
		if node.Id == "annotation-1" {
			foundWidget = node
			break
		}
	}
	require.NotNil(t, foundWidget, "should find widget in workflow nodes JSON")
	assert.Equal(t, componentpb.Node_TYPE_WIDGET, foundWidget.Type, "widget should have correct type")
	assert.Equal(t, "annotation", foundWidget.Widget.Name, "widget should have correct name")
	assert.Equal(t, annotationText, foundWidget.Configuration.AsMap()["text"], "widget text should match")

	updatedAnnotationText := "This is an updated annotation"
	updatedAnnotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": updatedAnnotationText,
	})

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:            "annotation-1",
					Name:          "Workflow Note Updated",
					Type:          componentpb.Node_TYPE_WIDGET,
					Configuration: updatedAnnotationConfig,
					Widget: &componentpb.Node_WidgetRef{
						Name: "annotation",
					},
				},
				{
					Id:   "component-1",
					Name: "Regular Component",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	finalUpdatedWorkflow, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed when updating widget nodes")

	// Verify updated widget is in JSON
	var updatedWidget *componentpb.Node
	for _, node := range finalUpdatedWorkflow.Workflow.Spec.Nodes {
		if node.Id == "annotation-1" {
			updatedWidget = node
			break
		}
	}
	require.NotNil(t, updatedWidget, "should find updated widget in workflow nodes JSON")
	assert.Equal(t, "Workflow Note Updated", updatedWidget.Name, "widget name should be updated")
	assert.Equal(t, updatedAnnotationText, updatedWidget.Configuration.AsMap()["text"], "widget text should be updated")
}

func TestUpdateWorkflow_WidgetNodesCannotConnect(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	annotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": "This is an annotation",
	})

	workflowWithWidgetAsSource := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:            "annotation-1",
					Name:          "Annotation Note",
					Type:          componentpb.Node_TYPE_WIDGET,
					Configuration: annotationConfig,
					Widget: &componentpb.Node_WidgetRef{
						Name: "annotation",
					},
				},
				{
					Id:   "component-1",
					Name: "Component",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
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

	_, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		workflowWithWidgetAsSource,
		"http://localhost:3000/api/v1",
	)
	require.Error(t, err, "UpdateWorkflow should fail when widget node is used as source")
	assert.Contains(t, err.Error(), "widget nodes cannot be used as source nodes")

	workflowWithWidgetAsTarget := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:   "component-1",
					Name: "Component",
					Type: componentpb.Node_TYPE_COMPONENT,
					Component: &componentpb.Node_ComponentRef{
						Name: "noop",
					},
				},
				{
					Id:            "annotation-1",
					Name:          "Annotation Note",
					Type:          componentpb.Node_TYPE_WIDGET,
					Configuration: annotationConfig,
					Widget: &componentpb.Node_WidgetRef{
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

	_, err = UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		workflowWithWidgetAsTarget,
		"http://localhost:3000/api/v1",
	)
	require.Error(t, err, "UpdateWorkflow should fail when widget node is used as target")
	assert.Contains(t, err.Error(), "widget nodes cannot be used as target nodes")
}

func TestUpdateWorkflow_WidgetTextLengthValidation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	// Test text exceeding max length (5001 characters)
	longText := strings.Repeat("a", 5001)
	longAnnotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": longText,
	})

	workflowWithLongAnnotation := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:            "annotation-1",
					Name:          "Long Annotation",
					Type:          componentpb.Node_TYPE_WIDGET,
					Configuration: longAnnotationConfig,
					Widget: &componentpb.Node_WidgetRef{
						Name: "annotation",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	response, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		workflowWithLongAnnotation,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err)

	var annotationNode *componentpb.Node
	for _, node := range response.Workflow.Spec.Nodes {
		if node.Id == "annotation-1" {
			annotationNode = node
			break
		}
	}
	require.NotNil(t, annotationNode)
	require.Equal(t, annotationNode.ErrorMessage, "field 'text': must be at most 5000 characters")

	// Test text at exactly max length (5000 characters)
	exactlyMaxText := strings.Repeat("b", 5000)
	maxAnnotationConfig, _ := structpb.NewStruct(map[string]interface{}{
		"text": exactlyMaxText,
	})

	workflowWithMaxAnnotation := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:            "annotation-2",
					Name:          "Max Length Annotation",
					Type:          componentpb.Node_TYPE_WIDGET,
					Configuration: maxAnnotationConfig,
					Widget: &componentpb.Node_WidgetRef{
						Name: "annotation",
					},
				},
			},
			Edges: []*componentpb.Edge{},
		},
	}

	maxUpdatedWorkflow, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		workflowWithMaxAnnotation,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed with max length annotation text")

	// Verify the max length annotation is in JSON
	var maxFoundAnnotation *componentpb.Node
	for _, node := range maxUpdatedWorkflow.Workflow.Spec.Nodes {
		if node.Id == "annotation-2" {
			maxFoundAnnotation = node
			break
		}
	}
	require.NotNil(t, maxFoundAnnotation, "should find max annotation in workflow nodes JSON")
	assert.Equal(t, exactlyMaxText, maxFoundAnnotation.Configuration.AsMap()["text"], "annotation text should match")

	// Widgets should NOT be persisted in workflow_nodes table
	var annotationNodeCount int64
	database.Conn().Model(&models.WorkflowNode{}).Where("workflow_id = ? AND node_id = ?", workflow.ID, "annotation-2").Count(&annotationNodeCount)
	assert.Equal(t, int64(0), annotationNodeCount, "widget nodes should not be persisted in workflow_nodes table")
}
