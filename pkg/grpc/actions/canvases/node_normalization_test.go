package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestNormalizeCanvasNodesWithoutGroups(t *testing.T) {
	t.Run("removes group widgets and flattens nested positions", func(t *testing.T) {
		nodes := []models.Node{
			{
				ID:       "outer-group",
				Name:     "Outer Group",
				Type:     models.NodeTypeWidget,
				Ref:      models.NodeRef{Widget: &models.WidgetRef{Name: groupWidgetName}},
				Position: models.Position{X: 100, Y: 200},
				Configuration: map[string]any{
					"childNodeIds": []any{"inner-group"},
				},
			},
			{
				ID:       "inner-group",
				Name:     "Inner Group",
				Type:     models.NodeTypeWidget,
				Ref:      models.NodeRef{Widget: &models.WidgetRef{Name: groupWidgetName}},
				Position: models.Position{X: 20, Y: 30},
				Configuration: map[string]any{
					"childNodeIds": []any{"component-1"},
				},
			},
			{
				ID:       "component-1",
				Name:     "Component 1",
				Type:     models.NodeTypeComponent,
				Position: models.Position{X: 5, Y: 6},
			},
		}

		normalized := normalizeCanvasNodesWithoutGroups(nodes)

		if assert.Len(t, normalized, 1) {
			assert.Equal(t, "component-1", normalized[0].ID)
			assert.Equal(t, models.Position{X: 125, Y: 236}, normalized[0].Position)
		}
	})

	t.Run("removes group widgets without ids", func(t *testing.T) {
		nodes := []models.Node{
			{
				Name:     "Legacy Group",
				Type:     models.NodeTypeWidget,
				Ref:      models.NodeRef{Widget: &models.WidgetRef{Name: groupWidgetName}},
				Position: models.Position{X: 50, Y: 60},
			},
			{
				ID:       "component-1",
				Name:     "Component 1",
				Type:     models.NodeTypeComponent,
				Position: models.Position{X: 5, Y: 6},
			},
		}

		normalized := normalizeCanvasNodesWithoutGroups(nodes)

		if assert.Len(t, normalized, 1) {
			assert.Equal(t, "component-1", normalized[0].ID)
			assert.Equal(t, models.Position{X: 5, Y: 6}, normalized[0].Position)
		}
	})
}
