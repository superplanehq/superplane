package canvases

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

func Test__DeleteCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, "invalid-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas is soft deleted, data remains until cleanup", func(t *testing.T) {
		//
		// Create a canvas with nodes, events, and executions
		//
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
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

		event1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		event2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event1.ID, event2.ID, nil)
		support.CreateQueueItem(t, canvas.ID, "node-1", event1.ID, event2.ID)

		//
		// Verify canvas and all canvas data exist before deletion
		//
		_, err := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		nodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
		support.VerifyCanvasEventsCount(t, canvas.ID, 2)
		support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
		support.VerifyNodeQueueCount(t, canvas.ID, 1)

		//
		// Delete the canvas (soft delete).
		//
		_, err = DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, canvas.ID.String())
		require.NoError(t, err)

		//
		// Verify canvas is soft deleted but associated data still exists.
		// The canvas should not be found via regular queries (soft delete).
		//
		_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// But the workflow should still exist when queried with Unscoped
		var canvasUnscoped models.Canvas
		err = database.Conn().Unscoped().Where("id = ?", canvas.ID).First(&canvasUnscoped).Error
		require.NoError(t, err)
		assert.NotNil(t, canvasUnscoped.DeletedAt)

		// Verify the name has been updated with deleted timestamp suffix
		assert.Contains(t, canvasUnscoped.Name, "(deleted-")
		assert.NotEqual(t, canvas.Name, canvasUnscoped.Name)

		// Associated data should still exist (cleanup worker handles this)
		nodes, err = models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
		support.VerifyCanvasEventsCount(t, canvas.ID, 2)
		support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
		support.VerifyNodeQueueCount(t, canvas.ID, 1)
	})

	t.Run("canvas node webhook remains until cleanup worker processes it", func(t *testing.T) {
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
		// Create a canvas with node that has webhook
		//
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
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
		// Delete the canvas (soft delete).
		//
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, canvas.ID.String())
		require.NoError(t, err)

		//
		// Verify canvas is soft deleted but webhook still exists.
		// The cleanup worker will handle webhook deletion later.
		//
		_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// Webhook should still exist since cleanup worker hasn't run
		_, err = models.FindWebhook(webhookID)
		require.NoError(t, err)
	})
}
