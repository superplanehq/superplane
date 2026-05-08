package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__CanvasMemoryContext(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	ctx := NewCanvasMemoryContext(database.Conn(), canvas.ID)

	t.Run("rejects empty namespace", func(t *testing.T) {
		assert.Error(t, ctx.Add("", map[string]any{"k": "v"}))

		_, err := ctx.Find("", map[string]any{"k": "v"})
		assert.Error(t, err)

		_, err = ctx.FindFirst("", map[string]any{"k": "v"})
		assert.Error(t, err)

		_, err = ctx.FindAll("")
		assert.Error(t, err)

		_, err = ctx.FindFirstInNamespace("")
		assert.Error(t, err)

		_, err = ctx.Delete("", map[string]any{"k": "v"})
		assert.Error(t, err)

		_, err = ctx.Update("", map[string]any{"k": "v"}, map[string]any{"x": 1})
		assert.Error(t, err)
	})

	t.Run("CRUD and find helpers", func(t *testing.T) {
		err := ctx.Add("machines", map[string]any{"sandbox_id": "a", "creator": "alice"})
		require.NoError(t, err)
		err = ctx.Add("machines", map[string]any{"sandbox_id": "b", "creator": "bob"})
		require.NoError(t, err)

		all, err := ctx.FindAll("machines")
		require.NoError(t, err)
		require.Len(t, all, 2)

		latest, err := ctx.FindFirstInNamespace("machines")
		require.NoError(t, err)
		require.Equal(t, map[string]any{"sandbox_id": "b", "creator": "bob"}, latest)

		matched, err := ctx.Find("machines", map[string]any{"sandbox_id": "a"})
		require.NoError(t, err)
		require.Equal(t, []any{map[string]any{"sandbox_id": "a", "creator": "alice"}}, matched)

		first, err := ctx.FindFirst("machines", map[string]any{"sandbox_id": "a"})
		require.NoError(t, err)
		require.Equal(t, map[string]any{"sandbox_id": "a", "creator": "alice"}, first)

		updated, err := ctx.Update("machines", map[string]any{"sandbox_id": "b"}, map[string]any{"patched": true})
		require.NoError(t, err)
		require.Len(t, updated, 1)

		deleted, err := ctx.Delete("machines", map[string]any{"sandbox_id": "a"})
		require.NoError(t, err)
		require.Len(t, deleted, 1)

		remaining, err := ctx.FindAll("machines")
		require.NoError(t, err)
		require.Len(t, remaining, 1)
	})

	t.Run("findFirst and findFirstInNamespace when missing", func(t *testing.T) {
		missing, err := ctx.FindFirst("empty-ns", map[string]any{"id": "none"})
		require.NoError(t, err)
		assert.Nil(t, missing)

		none, err := ctx.FindFirstInNamespace("empty-ns-other")
		require.NoError(t, err)
		assert.Nil(t, none)
	})
}
