package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test_UpdateNodePause(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

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
		nil,
	)

	t.Run("pauses and resumes node processing", func(t *testing.T) {
		response, err := UpdateNodePause(context.Background(), r.Registry, workflow.ID.String(), "node-1", true)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Node)
		assert.True(t, response.Node.Paused)

		node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, "node-1")
		require.NoError(t, err)
		assert.Equal(t, models.WorkflowNodeStatePaused, node.State)

		response, err = UpdateNodePause(context.Background(), r.Registry, workflow.ID.String(), "node-1", false)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Node)
		assert.False(t, response.Node.Paused)

		node, err = models.FindWorkflowNode(database.Conn(), workflow.ID, "node-1")
		require.NoError(t, err)
		assert.Equal(t, models.WorkflowNodeStateReady, node.State)
	})

	t.Run("resumes to processing when execution is running", func(t *testing.T) {
		rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		execution := support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, event.ID, nil)
		require.NoError(t, database.Conn().
			Model(execution).
			Update("state", models.WorkflowNodeExecutionStateStarted).
			Error)
		require.NoError(t, database.Conn().
			Model(&models.WorkflowNode{}).
			Where("workflow_id = ? AND node_id = ?", workflow.ID, "node-1").
			Update("state", models.WorkflowNodeStatePaused).
			Error)

		response, err := UpdateNodePause(context.Background(), r.Registry, workflow.ID.String(), "node-1", false)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Node)
		assert.False(t, response.Node.Paused)

		node, err := models.FindWorkflowNode(database.Conn(), workflow.ID, "node-1")
		require.NoError(t, err)
		assert.Equal(t, models.WorkflowNodeStateProcessing, node.State)
	})

	t.Run("invalid node type returns error", func(t *testing.T) {
		triggerWorkflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "trigger-1",
					Name:   "Trigger Node",
					Type:   models.NodeTypeTrigger,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Trigger: &models.TriggerRef{Name: "noop"},
					}),
				},
			},
			nil,
		)

		_, err := UpdateNodePause(context.Background(), r.Registry, triggerWorkflow.ID.String(), "trigger-1", true)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})
}
