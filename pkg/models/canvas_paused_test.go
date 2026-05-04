package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__Canvas_Paused(t *testing.T) {
	r := support.Setup(t)

	t.Run("FindWebhookNodesInTransaction excludes nodes from paused canvases", func(t *testing.T) {
		webhookID := uuid.New()
		webhook := &models.Webhook{
			ID:    webhookID,
			State: models.WebhookStateProvisioning,
		}
		require.NoError(t, database.Conn().Create(webhook).Error)

		// Create a paused canvas with a webhook node
		canvas, nodes := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID:    "node-1",
					Name:      "Webhook Trigger",
					Type:      models.NodeTypeTrigger,
					WebhookID: &webhookID,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Trigger: &models.TriggerRef{Name: "webhook"},
					}),
				},
			},
			[]models.Edge{},
		)

		// Pause the canvas
		require.NoError(t, database.Conn().Model(canvas).Update("paused", true).Error)

		// Check if the node is found
		foundNodes, err := models.FindWebhookNodesInTransaction(database.Conn(), webhookID)
		require.NoError(t, err)
		assert.Empty(t, foundNodes, "should not find nodes from paused canvas")

		// Resume the canvas
		require.NoError(t, database.Conn().Model(canvas).Update("paused", false).Error)

		// Check if the node is found now
		foundNodes, err = models.FindWebhookNodesInTransaction(database.Conn(), webhookID)
		require.NoError(t, err)
		assert.Len(t, foundNodes, 1)
		assert.Equal(t, nodes[0].NodeID, foundNodes[0].NodeID)
		assert.Equal(t, nodes[0].WorkflowID, foundNodes[0].WorkflowID)
	})

	t.Run("ListReadyTriggers excludes nodes from paused canvases", func(t *testing.T) {
		// Create a paused canvas with a ready trigger node
		_, nodes := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-2",
					Name:   "Ready Trigger",
					Type:   models.NodeTypeTrigger,
					State:  models.CanvasNodeStateReady,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Trigger: &models.TriggerRef{Name: "schedule"},
					}),
				},
			},
			[]models.Edge{},
		)

		// Find the canvas to pause it (CreateCanvas returns models.Canvas)
		var canvas models.Canvas
		require.NoError(t, database.Conn().First(&canvas, "id = ?", nodes[0].WorkflowID).Error)

		// Pause the canvas
		require.NoError(t, database.Conn().Model(&canvas).Update("paused", true).Error)

		// Check if the node is found
		readyTriggers, err := models.ListReadyTriggers()
		require.NoError(t, err)
		
		found := false
		for _, node := range readyTriggers {
			if node.NodeID == nodes[0].NodeID && node.WorkflowID == nodes[0].WorkflowID {
				found = true
				break
			}
		}
		assert.False(t, found, "should not find ready trigger from paused canvas")

		// Resume the canvas
		require.NoError(t, database.Conn().Model(&canvas).Update("paused", false).Error)

		// Check if the node is found now
		readyTriggers, err = models.ListReadyTriggers()
		require.NoError(t, err)
		
		found = false
		for _, node := range readyTriggers {
			if node.NodeID == nodes[0].NodeID && node.WorkflowID == nodes[0].WorkflowID {
				found = true
				break
			}
		}
		assert.True(t, found, "should find ready trigger from resumed canvas")
	})

	t.Run("ListNodeRequests excludes requests from paused canvases", func(t *testing.T) {
		// Create a canvas with a node
		canvas, nodes := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-3",
					Name:   "Task Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		// Create a pending request for this node
		request := &models.CanvasNodeRequest{
			ID:         uuid.New(),
			WorkflowID: canvas.ID,
			NodeID:     nodes[0].NodeID,
			State:      models.NodeExecutionRequestStatePending,
		}
		require.NoError(t, database.Conn().Create(request).Error)

		// Pause the canvas
		require.NoError(t, database.Conn().Model(canvas).Update("paused", true).Error)

		// Check if the request is found
		foundRequests, err := models.ListNodeRequests()
		require.NoError(t, err)
		
		found := false
		for _, req := range foundRequests {
			if req.ID == request.ID {
				found = true
				break
			}
		}
		assert.False(t, found, "should not find request from paused canvas")

		// Resume the canvas
		require.NoError(t, database.Conn().Model(canvas).Update("paused", false).Error)

		// Check if the request is found now
		foundRequests, err = models.ListNodeRequests()
		require.NoError(t, err)
		
		found = false
		for _, req := range foundRequests {
			if req.ID == request.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "should find request from resumed canvas")
	})
}
