package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestParseStagedSnapshotGraph(t *testing.T) {
	t.Run("reads staged action nodes without importing pkg/yaml", func(t *testing.T) {
		graph, err := parseStagedSnapshotGraph(`apiVersion: v1
kind: Canvas
metadata:
  name: edit-canvas
spec:
  nodes:
    - id: edit-node
      name: Edit HTTP
      type: TYPE_ACTION
      component: http
  edges:
    - sourceId: edit-node
      targetId: other
      channel: default
`)
		require.NoError(t, err)
		assert.Equal(t, "staging", graph.source)
		require.Len(t, graph.nodes, 1)
		assert.Equal(t, "edit-node", graph.nodes[0].ID)
		assert.Equal(t, "Edit HTTP", graph.nodes[0].Name)
		assert.Equal(t, models.NodeTypeComponent, graph.nodes[0].Type)
		require.NotNil(t, graph.nodes[0].Ref.Component)
		assert.Equal(t, "http", graph.nodes[0].Ref.Component.Name)
		require.Len(t, graph.edges, 1)
		assert.Equal(t, "edit-node", graph.edges[0].SourceID)
	})

	t.Run("maps trigger and widget refs", func(t *testing.T) {
		graph, err := parseStagedSnapshotGraph(`spec:
  nodes:
    - id: t1
      name: GitHub
      type: TYPE_TRIGGER
      component: github
    - id: w1
      name: Board
      type: TYPE_WIDGET
      component: board
  edges: []
`)
		require.NoError(t, err)
		require.Len(t, graph.nodes, 2)
		assert.Equal(t, models.NodeTypeTrigger, graph.nodes[0].Type)
		require.NotNil(t, graph.nodes[0].Ref.Trigger)
		assert.Equal(t, "github", graph.nodes[0].Ref.Trigger.Name)
		assert.Equal(t, models.NodeTypeWidget, graph.nodes[1].Type)
		require.NotNil(t, graph.nodes[1].Ref.Widget)
		assert.Equal(t, "board", graph.nodes[1].Ref.Widget.Name)
	})

	t.Run("rejects yaml with no spec", func(t *testing.T) {
		_, err := parseStagedSnapshotGraph("apiVersion: v1\nkind: Canvas\n")
		require.Error(t, err)
	})
}
