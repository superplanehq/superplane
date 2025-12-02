package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__ListNodeExecutions(t *testing.T) {
	r := support.Setup(t)

	t.Run("node does not exist -> 404 error", func(t *testing.T) {
		//
		// Create a workflow with a node
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Try to list executions for a non-existent node
		//
		_, err := ListNodeExecutions(
			context.Background(),
			r.Registry,
			workflow.ID.String(),
			"non-existent-node",
			[]pb.WorkflowNodeExecution_State{},
			[]pb.WorkflowNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify we get a NotFound error
		//
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "workflow node not found")
	})

	t.Run("workflow does not exist -> 404 error", func(t *testing.T) {
		//
		// Try to list executions for a non-existent workflow
		//
		_, err := ListNodeExecutions(
			context.Background(),
			r.Registry,
			uuid.New().String(),
			"some-node",
			[]pb.WorkflowNodeExecution_State{},
			[]pb.WorkflowNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify we get a NotFound error
		//
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "workflow node not found")
	})

	t.Run("returns executions for existing node", func(t *testing.T) {
		//
		// Create a workflow with a node
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		//
		// Create events and executions
		//
		rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, event.ID, nil)

		//
		// List executions for the node
		//
		response, err := ListNodeExecutions(
			context.Background(),
			r.Registry,
			workflow.ID.String(),
			"node-1",
			[]pb.WorkflowNodeExecution_State{},
			[]pb.WorkflowNodeExecution_Result{},
			0,
			nil,
		)

		//
		// Verify successful response
		//
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Executions, 1)
		assert.Equal(t, uint32(1), response.TotalCount)
		assert.Equal(t, "node-1", response.Executions[0].NodeId)
	})
}
