package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__ListGrew(t *testing.T) {
	t.Run("item added", func(t *testing.T) {
		changes := map[string]any{
			"labels": map[string]any{
				"previous": []any{map[string]any{"id": float64(1)}},
				"current":  []any{map[string]any{"id": float64(1)}, map[string]any{"id": float64(2)}},
			},
		}
		assert.True(t, listGrew(changes, "labels", "id"))
	})

	t.Run("no addition", func(t *testing.T) {
		changes := map[string]any{
			"labels": map[string]any{
				"previous": []any{map[string]any{"id": float64(1)}},
				"current":  []any{map[string]any{"id": float64(1)}},
			},
		}
		assert.False(t, listGrew(changes, "labels", "id"))
	})

	t.Run("scalar list", func(t *testing.T) {
		changes := map[string]any{
			"reviewer_ids": map[string]any{
				"previous": []any{float64(1)},
				"current":  []any{float64(1), float64(2)},
			},
		}
		assert.True(t, listGrew(changes, "reviewer_ids", ""))
	})

	t.Run("field missing", func(t *testing.T) {
		assert.False(t, listGrew(map[string]any{}, "labels", "id"))
	})
}

func Test__ListShrank(t *testing.T) {
	t.Run("item removed", func(t *testing.T) {
		changes := map[string]any{
			"assignees": map[string]any{
				"previous": []any{map[string]any{"id": float64(1)}, map[string]any{"id": float64(2)}},
				"current":  []any{map[string]any{"id": float64(1)}},
			},
		}
		assert.True(t, listShrank(changes, "assignees", "id"))
	})

	t.Run("no removal", func(t *testing.T) {
		changes := map[string]any{
			"assignees": map[string]any{
				"previous": []any{map[string]any{"id": float64(1)}},
				"current":  []any{map[string]any{"id": float64(1)}, map[string]any{"id": float64(2)}},
			},
		}
		assert.False(t, listShrank(changes, "assignees", "id"))
	})
}

func Test__ChangedFromNilToValue(t *testing.T) {
	assert.True(t, changedFromNilToValue(map[string]any{
		"milestone_id": map[string]any{"previous": nil, "current": float64(1)},
	}, "milestone_id"))

	assert.False(t, changedFromNilToValue(map[string]any{
		"milestone_id": map[string]any{"previous": float64(1), "current": float64(2)},
	}, "milestone_id"))

	assert.False(t, changedFromNilToValue(map[string]any{}, "milestone_id"))
}

func Test__ChangedFromValueToNil(t *testing.T) {
	assert.True(t, changedFromValueToNil(map[string]any{
		"milestone_id": map[string]any{"previous": float64(1), "current": nil},
	}, "milestone_id"))

	assert.False(t, changedFromValueToNil(map[string]any{
		"milestone_id": map[string]any{"previous": nil, "current": float64(1)},
	}, "milestone_id"))
}

func Test__ChangedBoolTo(t *testing.T) {
	changes := map[string]any{
		"draft": map[string]any{"previous": true, "current": false},
	}
	assert.True(t, changedBoolTo(changes, "draft", false))
	assert.False(t, changedBoolTo(changes, "draft", true))
	assert.False(t, changedBoolTo(changes, "missing", false))
}

func Test__ChangedField(t *testing.T) {
	changes := map[string]any{
		"title": map[string]any{"previous": "old", "current": "new"},
	}
	assert.True(t, changedField(changes, "title"))
	assert.False(t, changedField(changes, "description"))
}
