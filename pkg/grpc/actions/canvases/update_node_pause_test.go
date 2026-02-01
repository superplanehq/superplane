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

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
		response, err := UpdateNodePause(context.Background(), r.Registry, canvas.ID.String(), "node-1", true)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Node)
		assert.True(t, response.Node.Paused)

		node, err := models.FindCanvasNode(database.Conn(), canvas.ID, "node-1")
		require.NoError(t, err)
		assert.Equal(t, models.CanvasNodeStatePaused, node.State)

		response, err = UpdateNodePause(context.Background(), r.Registry, canvas.ID.String(), "node-1", false)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Node)
		assert.False(t, response.Node.Paused)

		node, err = models.FindCanvasNode(database.Conn(), canvas.ID, "node-1")
		require.NoError(t, err)
		assert.Equal(t, models.CanvasNodeStateReady, node.State)
	})

	t.Run("resumes to processing when execution is running", func(t *testing.T) {
		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, event.ID, nil)
		require.NoError(t, database.Conn().
			Model(execution).
			Update("state", models.CanvasNodeExecutionStateStarted).
			Error)
		require.NoError(t, database.Conn().
			Model(&models.CanvasNode{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, "node-1").
			Update("state", models.CanvasNodeStatePaused).
			Error)

		response, err := UpdateNodePause(context.Background(), r.Registry, canvas.ID.String(), "node-1", false)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Node)
		assert.False(t, response.Node.Paused)

		node, err := models.FindCanvasNode(database.Conn(), canvas.ID, "node-1")
		require.NoError(t, err)
		assert.Equal(t, models.CanvasNodeStateProcessing, node.State)
	})

	t.Run("invalid node type returns error", func(t *testing.T) {
		triggerCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
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

		_, err := UpdateNodePause(context.Background(), r.Registry, triggerCanvas.ID.String(), "trigger-1", true)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})
}
