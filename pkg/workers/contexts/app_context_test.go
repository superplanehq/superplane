package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__AppContext__Get(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	listenerCanvas, listenerNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "on-broadcast",
				Name:   "On Broadcast",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"app": "",
				}),
			},
		},
		nil,
	)

	sourceCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	ctx := NewAppContext(database.Conn(), listenerCanvas, &listenerNodes[0])

	t.Run("returns app by id", func(t *testing.T) {
		app, err := ctx.Get(sourceCanvas.ID.String())
		require.NoError(t, err)
		assert.Equal(t, sourceCanvas.ID.String(), app.ID)
		assert.Equal(t, sourceCanvas.Name, app.Name)
	})

	t.Run("returns app by name", func(t *testing.T) {
		app, err := ctx.Get(sourceCanvas.Name)
		require.NoError(t, err)
		assert.Equal(t, sourceCanvas.ID.String(), app.ID)
		assert.Equal(t, sourceCanvas.Name, app.Name)
	})

	t.Run("returns not found for missing app", func(t *testing.T) {
		app, err := ctx.Get(uuid.New().String())
		require.ErrorIs(t, err, core.ErrNotFound)
		assert.Nil(t, app)
	})
}

func Test__AppContext__Subscribe(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	listenerCanvas, listenerNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "on-broadcast",
				Name:   "On Broadcast",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"app": "",
				}),
			},
		},
		nil,
	)

	sourceCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	ctx := NewAppContext(database.Conn(), listenerCanvas, &listenerNodes[0])

	t.Run("creates subscription", func(t *testing.T) {
		err := ctx.Subscribe(sourceCanvas.ID.String())
		require.NoError(t, err)

		var sub models.CanvasSubscription
		err = database.Conn().
			Where("source_canvas_id = ? AND target_canvas_id = ? AND target_node_id = ?",
				sourceCanvas.ID, listenerCanvas.ID, listenerNodes[0].NodeID).
			First(&sub).
			Error
		require.NoError(t, err)
	})

	t.Run("subscribe is idempotent", func(t *testing.T) {
		err := ctx.Subscribe(sourceCanvas.Name)
		require.NoError(t, err)

		err = ctx.Subscribe(sourceCanvas.Name)
		require.NoError(t, err)

		var count int64
		err = database.Conn().
			Model(&models.CanvasSubscription{}).
			Where("source_canvas_id = ? AND target_canvas_id = ? AND target_node_id = ?",
				sourceCanvas.ID, listenerCanvas.ID, listenerNodes[0].NodeID).
			Count(&count).
			Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("returns not found for missing source app", func(t *testing.T) {
		err := ctx.Subscribe(uuid.New().String())
		require.ErrorIs(t, err, core.ErrNotFound)
	})
}
