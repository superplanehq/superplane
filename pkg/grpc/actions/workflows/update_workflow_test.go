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

func TestUpdateWorkflow_AnnotationNodesHandled(t *testing.T) {
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
	updatedAnnotationText := "This is an updated annotation"

	initialWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:             "annotation-1",
					Name:           "Workflow Note",
					Type:           componentpb.Node_TYPE_ANNOTATION,
					AnnotationText: annotationText,
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

	_, err := UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		initialWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed with annotation nodes")

	var annotationNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "annotation-1").First(&annotationNode).Error
	require.NoError(t, err, "should be able to find annotation node")
	assert.Equal(t, models.NodeTypeAnnotation, annotationNode.Type, "annotation node should have correct type")
	assert.Equal(t, models.WorkflowNodeStateStatic, annotationNode.State, "annotation node should be static")
	assert.Nil(t, annotationNode.StateReason, "annotation node should not have error reason")
	assert.NotNil(t, annotationNode.AnnotationText, "annotation node should have annotation text")
	assert.Equal(t, annotationText, *annotationNode.AnnotationText, "annotation text should match")

	var componentNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "component-1").First(&componentNode).Error
	require.NoError(t, err, "should be able to find component node")
	assert.Equal(t, models.NodeTypeComponent, componentNode.Type, "component node should have correct type")

	updatedWorkflowPB := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:             "annotation-1",
					Name:           "Workflow Note Updated",
					Type:           componentpb.Node_TYPE_ANNOTATION,
					AnnotationText: updatedAnnotationText,
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

	_, err = UpdateWorkflow(
		context.Background(),
		r.Encryptor,
		r.Registry,
		r.Organization.ID.String(),
		workflow.ID.String(),
		updatedWorkflowPB,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed when updating annotation nodes")

	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "annotation-1").First(&annotationNode).Error
	require.NoError(t, err, "should be able to find updated annotation node")
	assert.Equal(t, "Workflow Note Updated", annotationNode.Name, "annotation node name should be updated")
	assert.NotNil(t, annotationNode.AnnotationText, "annotation node should still have annotation text")
	assert.Equal(t, updatedAnnotationText, *annotationNode.AnnotationText, "annotation text should be updated")
}

func TestUpdateWorkflow_AnnotationNodesCannotConnect(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	workflowWithAnnotationAsSource := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:             "annotation-1",
					Name:           "Annotation Note",
					Type:           componentpb.Node_TYPE_ANNOTATION,
					AnnotationText: "This is an annotation",
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
		workflowWithAnnotationAsSource,
		"http://localhost:3000/api/v1",
	)
	require.Error(t, err, "UpdateWorkflow should fail when annotation node is used as source")
	assert.Contains(t, err.Error(), "annotation nodes cannot be used as source nodes")

	workflowWithAnnotationAsTarget := &pb.Workflow{
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
					Id:             "annotation-1",
					Name:           "Annotation Note",
					Type:           componentpb.Node_TYPE_ANNOTATION,
					AnnotationText: "This is an annotation",
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
		workflowWithAnnotationAsTarget,
		"http://localhost:3000/api/v1",
	)
	require.Error(t, err, "UpdateWorkflow should fail when annotation node is used as target")
	assert.Contains(t, err.Error(), "annotation nodes cannot be used as target nodes")
}

func TestUpdateWorkflow_AnnotationTextLengthValidation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{},
		[]models.Edge{},
	)

	longText := strings.Repeat("a", 5001)

	workflowWithLongAnnotation := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:             "annotation-1",
					Name:           "Long Annotation",
					Type:           componentpb.Node_TYPE_ANNOTATION,
					AnnotationText: longText,
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
		workflowWithLongAnnotation,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed with long annotation text (validation creates node with error)")

	var annotationNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "annotation-1").First(&annotationNode).Error
	require.NoError(t, err, "should be able to find annotation node")
	assert.Equal(t, models.WorkflowNodeStateError, annotationNode.State, "annotation node should be in error state")
	assert.NotNil(t, annotationNode.StateReason, "annotation node should have error reason")
	assert.Contains(t, *annotationNode.StateReason, "cannot exceed 5000 characters", "error should mention character limit")
	assert.Nil(t, annotationNode.AnnotationText, "annotation text should not be stored when there's a validation error")

	exactlyMaxText := strings.Repeat("b", 5000)

	workflowWithMaxAnnotation := &pb.Workflow{
		Metadata: &pb.Workflow_Metadata{
			Name:        workflow.Name,
			Description: workflow.Description,
		},
		Spec: &pb.Workflow_Spec{
			Nodes: []*componentpb.Node{
				{
					Id:             "annotation-2",
					Name:           "Max Length Annotation",
					Type:           componentpb.Node_TYPE_ANNOTATION,
					AnnotationText: exactlyMaxText,
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
		workflowWithMaxAnnotation,
		"http://localhost:3000/api/v1",
	)
	require.NoError(t, err, "UpdateWorkflow should succeed with max length annotation text")

	var maxAnnotationNode models.WorkflowNode
	err = database.Conn().Where("workflow_id = ? AND node_id = ?", workflow.ID, "annotation-2").First(&maxAnnotationNode).Error
	require.NoError(t, err, "should be able to find max annotation node")
	assert.Equal(t, models.WorkflowNodeStateStatic, maxAnnotationNode.State, "annotation node should be in static state")
	assert.Nil(t, maxAnnotationNode.StateReason, "annotation node should not have error reason")
	assert.NotNil(t, maxAnnotationNode.AnnotationText, "annotation node should have annotation text")
	assert.Equal(t, exactlyMaxText, *maxAnnotationNode.AnnotationText, "annotation text should match")
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
