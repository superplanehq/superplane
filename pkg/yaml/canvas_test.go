package yaml

import (
	"bytes"
	"encoding/json"
	"testing"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanvasFromYAML_ReadsUnquotedPositionY(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
spec:
  nodes:
    - id: node-1
      name: Positioned
      type: TYPE_ACTION
      component: noop
      position:
        x: 100
        y: 200
  edges: []
`)

	resource, err := CanvasFromYAML(raw)
	require.NoError(t, err)
	require.Len(t, resource.Spec.Nodes, 1)
	assert.Equal(t, 100, resource.Spec.Nodes[0].Position.X)
	assert.Equal(t, 200, resource.Spec.Nodes[0].Position.Y)
}

func TestCanvasFromYAML_NormalizesLegacyYAML11PositionKey(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
spec:
  nodes:
    - id: node-1
      name: "Positioned"
      type: TYPE_ACTION
      component: noop
      position:
        x: 100
        y: 200
  edges: []
`)

	jsonBytes, err := ghodssyaml.YAMLToJSON(raw)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &doc))
	normalizeCanvasDocument(doc)

	normalizedJSON, err := json.Marshal(doc)
	require.NoError(t, err)

	var resource Canvas
	decoder := json.NewDecoder(bytes.NewReader(normalizedJSON))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&resource))

	require.Len(t, resource.Spec.Nodes, 1)
	assert.Equal(t, 100, resource.Spec.Nodes[0].Position.X)
	assert.Equal(t, 200, resource.Spec.Nodes[0].Position.Y)
}

func TestNormalizeYAML1YKey(t *testing.T) {
	t.Run("maps true key to y", func(t *testing.T) {
		position := map[string]any{"x": 10, "true": 20}
		normalizeYAML1YKey(position)
		assert.Equal(t, 20, position["y"])
		assert.NotContains(t, position, "true")
	})

	t.Run("leaves existing y unchanged", func(t *testing.T) {
		position := map[string]any{"x": 10, "y": 20, "true": 99}
		normalizeYAML1YKey(position)
		assert.Equal(t, 20, position["y"])
		assert.Equal(t, 99, position["true"])
	})
}
