package contexts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__AppExecutionContext__Broadcast(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "broadcast-message",
				Name:          "Broadcast Message",
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	ctx := NewAppExecutionContext(database.Conn(), canvas, &nodes[0], nil)
	payload := map[string]any{"message": "hello"}

	require.NoError(t, ctx.Broadcast(payload))

	var message models.AppMessage
	err := database.Conn().
		Where("canvas_id = ? AND node_id = ?", canvas.ID, nodes[0].NodeID).
		First(&message).
		Error
	require.NoError(t, err)

	var storedPayload map[string]any
	require.NoError(t, json.Unmarshal(message.Payload, &storedPayload))
	assert.Equal(t, payload, storedPayload)
}
