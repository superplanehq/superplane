package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test_DeleteNodeQueueItem(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	nodeID := "component-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: nodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		nil,
	)

	// Create a single queue item on that node
	event := support.EmitWorkflowEventForNode(t, workflow.ID, nodeID, "default", nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, nodeID, event.ID, event.ID)

	items, err := models.ListNodeQueueItems(workflow.ID, nodeID, 10, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)

	_, err = DeleteNodeQueueItem(context.Background(), r.Registry, workflow.ID.String(), nodeID, items[0].ID.String())
	require.NoError(t, err)

	remaining, err := models.ListNodeQueueItems(workflow.ID, nodeID, 10, nil)
	require.NoError(t, err)
	require.Len(t, remaining, 0)
}
