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

func Test__ExecutionStateContext__Emit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
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
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	t.Run("rejects large payload", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), execution)
		largePayload := strings.Repeat("a", DefaultMaxPayloadSize+100)

		err := ctx.Emit("default", "test.payload", []any{largePayload})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event payload too large")
		support.VerifyCanvasNodeEventsCount(t, canvas.ID, componentNodeID, 0)
	})
}
