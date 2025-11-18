package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__DeleteWorkflow(t *testing.T) {
	r := support.Setup(t)

	t.Run("workflow does not exist -> error", func(t *testing.T) {
		_, err := DeleteWorkflow(context.Background(), r.Registry, r.Organization.ID, uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("invalid workflow id -> error", func(t *testing.T) {
		_, err := DeleteWorkflow(context.Background(), r.Registry, r.Organization.ID, "invalid-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("workflow is deleted along with nodes, events, and executions", func(t *testing.T) {
		//
		// Create a workflow with nodes, events, and executions
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "node-2",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		event1 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
		event2 := support.EmitWorkflowEventForNode(t, workflow.ID, "node-2", "default", nil)
		support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", event1.ID, event2.ID, nil)
		support.CreateWorkflowQueueItem(t, workflow.ID, "node-1", event1.ID, event2.ID)

		//
		// Verify workflow and all workflow data exist before deletion
		//
		_, err := models.FindWorkflow(r.Organization.ID, workflow.ID)
		require.NoError(t, err)
		nodes, err := models.FindWorkflowNodes(workflow.ID)
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
		support.VerifyWorkflowEventsCount(t, workflow.ID, 2)
		support.VerifyWorkflowNodeExecutionsCount(t, workflow.ID, 1)
		support.VerifyWorkflowNodeQueueCount(t, workflow.ID, 1)

		//
		// Delete the workflow.
		//
		_, err = DeleteWorkflow(context.Background(), r.Registry, r.Organization.ID, workflow.ID.String())
		require.NoError(t, err)

		//
		// Verify workflow and all associated data is deleted.
		//
		_, err = models.FindWorkflow(r.Organization.ID, workflow.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		nodes, err = models.FindWorkflowNodes(workflow.ID)
		require.NoError(t, err)
		assert.Len(t, nodes, 0)
		support.VerifyWorkflowEventsCount(t, workflow.ID, 0)
		support.VerifyWorkflowNodeExecutionsCount(t, workflow.ID, 0)
		support.VerifyWorkflowNodeQueueCount(t, workflow.ID, 0)
	})

	t.Run("workflow node webhook is deleted", func(t *testing.T) {
		//
		// Create webhook
		//
		webhookID := uuid.New()
		webhook := models.Webhook{
			ID:     webhookID,
			State:  models.WebhookStatePending,
			Secret: []byte("secret"),
		}

		require.NoError(t, database.Conn().Create(&webhook).Error)

		//
		// Create a workflow with node that has webhook
		//
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
					WebhookID: &webhookID,
				},
			},
			[]models.Edge{},
		)

		//
		// Delete the workflow, and verify webhook is deleted too.
		//
		_, err := DeleteWorkflow(context.Background(), r.Registry, r.Organization.ID, workflow.ID.String())
		require.NoError(t, err)
		_, err = models.FindWebhook(webhookID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
