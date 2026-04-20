package contexts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	_ "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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
			"value": strings.Repeat("a", DefaultMaxPayloadSize+100),
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

	t.Run("uses default run title from trigger definition", func(t *testing.T) {
		bitbucketCanvas, bitbucketNodes := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID:        triggerNodeID,
					Name:          triggerNodeID,
					Type:          models.NodeTypeTrigger,
					Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPush"}}),
					Configuration: datatypes.NewJSONType(map[string]any{}),
				},
			},
			nil,
		)

		ctx := NewEventContext(database.Conn(), &bitbucketNodes[0], nil)
		require.NoError(t, ctx.Emit("bitbucket.push", map[string]any{
			"repository": map[string]any{"full_name": "superplanehq/superplane"},
			"push": map[string]any{
				"changes": []any{
					map[string]any{
						"new": map[string]any{
							"target": map[string]any{"hash": "abcdef1234567890", "message": "Ship it"},
						},
					},
				},
			},
		}))

		events, err := models.ListCanvasEvents(bitbucketCanvas.ID, triggerNodeID, 10, nil)
		require.NoError(t, err)
		require.NotEmpty(t, events)
		require.NotNil(t, events[len(events)-1].RunTitle)
		assert.Equal(t, "Ship it", *events[len(events)-1].RunTitle)
	})
}
