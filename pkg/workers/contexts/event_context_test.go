package contexts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
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
		ctx := NewEventContext(database.Conn(), &nodes[0])
		largePayload := map[string]any{
			"value": strings.Repeat("a", DefaultMaxPayloadSize+100),
		}

		err := ctx.Emit("test.payload", largePayload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event payload too large")
		support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	})

	t.Run("stores continuation key when provided", func(t *testing.T) {
		ctx := NewEventContext(database.Conn(), &nodes[0])
		err := ctx.EmitWithContinuation("test.payload", map[string]any{"ok": true}, "github:owner/repo:pr:42")
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(canvas.ID, triggerNodeID, 10, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.NotNil(t, events[0].ContinuationKey)
		assert.Equal(t, "github:owner/repo:pr:42", *events[0].ContinuationKey)
	})
}
