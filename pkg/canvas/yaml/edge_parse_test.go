package yaml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCanvasResourceEdgeFieldNames(t *testing.T) {
	t.Run("camelCase sourceId and targetId", func(t *testing.T) {
		canvas, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Test
spec:
  nodes:
    - id: a
      name: A
      type: TYPE_TRIGGER
      component: start
    - id: b
      name: B
      type: TYPE_ACTION
      component: filter
  edges:
    - sourceId: a
      targetId: b
`))
		require.NoError(t, err)
		require.Len(t, canvas.Spec.Edges, 1)
		require.Equal(t, "a", canvas.Spec.Edges[0].SourceId)
		require.Equal(t, "b", canvas.Spec.Edges[0].TargetId)
	})

	t.Run("snake_case source_id and target_id", func(t *testing.T) {
		canvas, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Test
spec:
  nodes:
    - id: a
      name: A
      type: TYPE_TRIGGER
      component: start
    - id: b
      name: B
      type: TYPE_ACTION
      component: filter
  edges:
    - source_id: a
      target_id: b
`))
		require.NoError(t, err)
		require.Len(t, canvas.Spec.Edges, 1)
		require.Equal(t, "a", canvas.Spec.Edges[0].SourceId)
		require.Equal(t, "b", canvas.Spec.Edges[0].TargetId)
	})

	t.Run("omitted channel defaults during canvas parse", func(t *testing.T) {
		canvas, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Test
spec:
  nodes:
    - id: a
      name: A
      type: TYPE_TRIGGER
      component: start
    - id: b
      name: B
      type: TYPE_ACTION
      component: filter
  edges:
    - sourceId: a
      targetId: b
`))
		require.NoError(t, err)
		require.Len(t, canvas.Spec.Edges, 1)
		require.Equal(t, "", canvas.Spec.Edges[0].Channel)
	})

	t.Run("position y survives yaml 1.1 boolean alias", func(t *testing.T) {
		canvas, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Test
spec:
  nodes:
    - id: a
      name: A
      type: TYPE_ACTION
      component: noop
      position:
        x: 500
        y: 200
  edges: []
`))
		require.NoError(t, err)
		require.Len(t, canvas.Spec.Nodes, 1)
		require.NotNil(t, canvas.Spec.Nodes[0].Position)
		require.Equal(t, int32(200), canvas.Spec.Nodes[0].Position.Y)
	})
}
