package contexts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__EventContext__Emit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        triggerNodeID,
				Name:          triggerNodeID,
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	t.Run("rejects large payload", func(t *testing.T) {
		ctx := NewEventContext(database.Conn(), &nodes[0], nil)
		largePayload := map[string]any{
			"value": strings.Repeat("a", config.MaxPayloadSize()+100),
		}

		err := ctx.Emit("test.payload", largePayload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event payload too large")
		support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	})

	t.Run("uses callback", func(t *testing.T) {
		newEvents := []models.CanvasEvent{}
		onNewEvents := func(events []models.CanvasEvent) {
			newEvents = append(newEvents, events...)
		}

		ctx := NewEventContext(database.Conn(), &nodes[0], onNewEvents)
		require.NoError(t, ctx.Emit("test.payload", map[string]any{"n": 1}))
		require.NoError(t, ctx.Emit("test.payload", map[string]any{"n": 2}))
		assert.Len(t, newEvents, 2)
	})

	t.Run("links trigger event to parent run when _superplane.parentRunId is provided", func(t *testing.T) {
		parentEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
		var parentRun *models.CanvasRun
		require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
			var err error
			parentRun, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, parentEvent)
			return err
		}))

		ctx := NewEventContext(database.Conn(), &nodes[0], nil)
		require.NoError(t, ctx.Emit("test.payload", map[string]any{
			"value": "child",
			"_superplane": map[string]any{
				"parentRunId": parentRun.ID.String(),
			},
		}))

		events, err := models.ListRootCanvasEventsInTransaction(database.Conn(), canvas.ID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		run, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, events[0].RunID)
		require.NoError(t, err)
		require.NotNil(t, run.ParentRunID)
		assert.Equal(t, parentRun.ID, *run.ParentRunID)
	})
}
