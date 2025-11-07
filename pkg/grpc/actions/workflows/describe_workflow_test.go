package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__DescribeWorkflow(t *testing.T) {
	r := support.Setup(t)

	t.Run("workflow does not exist -> error", func(t *testing.T) {
		_, err := DescribeWorkflow(context.Background(), r.Registry, r.Organization.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("invalid workflow id -> error", func(t *testing.T) {
		_, err := DescribeWorkflow(context.Background(), r.Registry, r.Organization.ID.String(), "invalid-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("returns workflow with metadata/spec/status structure", func(t *testing.T) {
		//
		// Create a workflow with nodes and edges
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

		//
		// Describe the workflow
		//
		response, err := DescribeWorkflow(context.Background(), r.Registry, r.Organization.ID.String(), workflow.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Workflow)

		//
		// Verify metadata structure
		//
		require.NotNil(t, response.Workflow.Metadata)
		assert.Equal(t, workflow.ID.String(), response.Workflow.Metadata.Id)
		assert.Equal(t, workflow.OrganizationID.String(), response.Workflow.Metadata.OrganizationId)
		assert.Equal(t, workflow.Name, response.Workflow.Metadata.Name)
		assert.Equal(t, workflow.Description, response.Workflow.Metadata.Description)
		assert.NotNil(t, response.Workflow.Metadata.CreatedAt)
		assert.NotNil(t, response.Workflow.Metadata.UpdatedAt)
		assert.NotNil(t, response.Workflow.Metadata.CreatedBy)

		//
		// Verify spec structure
		//
		require.NotNil(t, response.Workflow.Spec)
		assert.Len(t, response.Workflow.Spec.Nodes, 2)
		assert.Equal(t, "node-1", response.Workflow.Spec.Nodes[0].Id)
		assert.Equal(t, "First Node", response.Workflow.Spec.Nodes[0].Name)
		assert.Equal(t, "node-2", response.Workflow.Spec.Nodes[1].Id)
		assert.Equal(t, "Second Node", response.Workflow.Spec.Nodes[1].Name)

		assert.Len(t, response.Workflow.Spec.Edges, 1)
		assert.Equal(t, "node-1", response.Workflow.Spec.Edges[0].SourceId)
		assert.Equal(t, "node-2", response.Workflow.Spec.Edges[0].TargetId)
		assert.Equal(t, "default", response.Workflow.Spec.Edges[0].Channel)
	})
}
