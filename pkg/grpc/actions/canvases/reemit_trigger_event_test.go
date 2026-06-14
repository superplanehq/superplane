package canvases

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ReemitTriggerEvent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "start-node"
	componentNodeID := "noop-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNodeID,
				Name:   componentNodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		nil,
	)

	ctx := context.Background()

	t.Run("canvas not found -> error", func(t *testing.T) {
		_, err := ReemitTriggerEvent(ctx, r.Organization.ID, uuid.New(), triggerNodeID, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canvas not found")
	})

	t.Run("trigger node not found -> error", func(t *testing.T) {
		_, err := ReemitTriggerEvent(ctx, r.Organization.ID, canvas.ID, "missing-node", uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node not found")
	})

	t.Run("node is not trigger -> error", func(t *testing.T) {
		_, err := ReemitTriggerEvent(ctx, r.Organization.ID, canvas.ID, componentNodeID, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a trigger")
	})

	t.Run("event not found -> error", func(t *testing.T) {
		_, err := ReemitTriggerEvent(ctx, r.Organization.ID, canvas.ID, triggerNodeID, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")
	})

	t.Run("event belongs to another node -> error", func(t *testing.T) {
		now := time.Now()
		sourceEvent := models.CanvasEvent{
			WorkflowID: canvas.ID,
			NodeID:     componentNodeID,
			Channel:    "default",
			Data:       models.NewJSONValue(map[string]any{"data": map[string]any{"message": "Hello"}}),
			State:      models.CanvasEventStatePending,
			CreatedAt:  &now,
		}
		require.NoError(t, database.Conn().Create(&sourceEvent).Error)

		_, err := ReemitTriggerEvent(ctx, r.Organization.ID, canvas.ID, triggerNodeID, sourceEvent.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong to trigger node")
	})

	t.Run("non-root event -> error", func(t *testing.T) {
		sourceEvent := models.CanvasEvent{
			WorkflowID: canvas.ID,
			NodeID:     triggerNodeID,
			Channel:    "default",
			Data:       models.NewJSONValue(map[string]any{"data": map[string]any{"message": "Hello"}}),
			State:      models.CanvasEventStatePending,
			CreatedAt:  ptr(time.Now()),
		}
		require.NoError(t, database.Conn().Create(&sourceEvent).Error)

		execution := support.CreateCanvasNodeExecution(
			t,
			canvas.ID,
			triggerNodeID,
			sourceEvent.ID,
			sourceEvent.ID,
		)
		require.NoError(
			t,
			database.Conn().
				Model(&models.CanvasEvent{}).
				Where("id = ?", sourceEvent.ID).
				Update("execution_id", execution.ID).
				Error,
		)

		_, err := ReemitTriggerEvent(ctx, r.Organization.ID, canvas.ID, triggerNodeID, sourceEvent.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "root trigger events")
	})

	t.Run("successfully re-emits trigger root event", func(t *testing.T) {
		customName := "Run: hello"
		now := time.Now()
		sourceEvent := models.CanvasEvent{
			WorkflowID: canvas.ID,
			NodeID:     triggerNodeID,
			Channel:    "default",
			Data: models.NewJSONValue(map[string]any{
				"type":      "manual.run",
				"timestamp": now.UTC().Format(time.RFC3339Nano),
				"data":      map[string]any{"message": "Hello"},
			}),
			CustomName: &customName,
			State:      models.CanvasEventStatePending,
			CreatedAt:  &now,
		}
		require.NoError(t, database.Conn().Create(&sourceEvent).Error)

		response, err := ReemitTriggerEvent(ctx, r.Organization.ID, canvas.ID, triggerNodeID, sourceEvent.ID)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotEmpty(t, response.EventId)
		assert.NotEqual(t, sourceEvent.ID.String(), response.EventId)

		reemittedID, err := uuid.Parse(response.EventId)
		require.NoError(t, err)

		reemitted, err := models.FindCanvasEvent(reemittedID)
		require.NoError(t, err)
		assert.Equal(t, canvas.ID, reemitted.WorkflowID)
		assert.Equal(t, triggerNodeID, reemitted.NodeID)
		assert.Equal(t, sourceEvent.Channel, reemitted.Channel)
		assert.Equal(t, sourceEvent.Data.Data(), reemitted.Data.Data())
		assert.Equal(t, models.CanvasEventStatePending, reemitted.State)
		require.NotNil(t, reemitted.CustomName)
		assert.Equal(t, customName, *reemitted.CustomName)
	})
}

func ptr[T any](v T) *T {
	return &v
}
