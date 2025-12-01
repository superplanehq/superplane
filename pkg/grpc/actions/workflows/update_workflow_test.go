package workflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestUpdateWorkflow_NodeRemovalUseSoftDelete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, nodes := support.CreateWorkflow(
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

	existingNodes := []models.WorkflowNode{nodes[0], nodes[1]}
	newNodes := []models.Node{{ID: "node-1"}}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return deleteNodes(tx, existingNodes, newNodes, workflow.ID)
	})
	require.NoError(t, err, "deleteNodes should succeed when removing nodes with execution KVs")

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
