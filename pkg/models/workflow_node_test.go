package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestDeleteWorkflowNode_WithExecutionKVs(t *testing.T) {
	db := database.Conn()

	organizationID := uuid.New()
	workflowID := uuid.New()
	nodeID := "test-node"
	executionID := uuid.New()
	now := time.Now()

	workflow := Workflow{
		ID:             workflowID,
		OrganizationID: organizationID,
		Name:           "Test Workflow",
		Description:    "Test workflow for node deletion",
		Edges:          datatypes.NewJSONSlice([]Edge{}),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, db.Create(&workflow).Error)
	defer db.Unscoped().Delete(&workflow)

	node := WorkflowNode{
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		Name:          "Test Node",
		State:         WorkflowNodeStateReady,
		Type:          NodeTypeComponent,
		Ref:           datatypes.NewJSONType(NodeRef{}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(Position{}),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}

	require.NoError(t, db.Create(&node).Error)
	defer db.Unscoped().Delete(&node)

	event := WorkflowEvent{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType[any](map[string]any{}),
		State:      WorkflowEventStatePending,
		CreatedAt:  &now,
	}

	require.NoError(t, db.Create(&event).Error)
	defer db.Unscoped().Delete(&event)

	execution := WorkflowNodeExecution{
		ID:          executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		RootEventID: event.ID,
		EventID:     event.ID,
		State:       WorkflowNodeExecutionStatePending,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	require.NoError(t, db.Create(&execution).Error)
	defer db.Unscoped().Delete(&execution)

	kv := WorkflowNodeExecutionKV{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		ExecutionID: executionID,
		Key:         "test-key",
		Value:       "test-value",
		CreatedAt:   &now,
	}

	require.NoError(t, db.Create(&kv).Error)
	defer db.Unscoped().Delete(&kv)

	err := db.Transaction(func(tx *gorm.DB) error {
		return DeleteWorkflowNode(tx, node)
	})

	assert.NoError(t, err, "DeleteWorkflowNode should handle execution KV cleanup")

	var nodeCount int64
	db.Model(&WorkflowNode{}).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&nodeCount)
	assert.Equal(t, int64(0), nodeCount, "Node should be deleted")

	var kvCount int64
	db.Model(&WorkflowNodeExecutionKV{}).Where("workflow_id = ? AND node_id = ?", workflowID, nodeID).Count(&kvCount)
	assert.Equal(t, int64(0), kvCount, "Execution KV should be cleaned up")
}
