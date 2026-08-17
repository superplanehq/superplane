package yaml

import (
	"bytes"
	"encoding/json"
	"testing"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestCanvasFromYAML_ReadsUnquotedPositionY(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: test
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

func TestVersionToCanvasYAML_OmitsEmptyIntegrationName(t *testing.T) {
	integrationID := "11111111-1111-1111-1111-111111111111"
	version := &models.CanvasVersion{
		Nodes: []models.Node{
			{
				ID:            "node-1",
				Name:          "Deploy",
				Type:          models.NodeTypeComponent,
				Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
				IntegrationID: &integrationID,
			},
		},
	}

	out, err := VersionToCanvasYAML("canvas", "", version)
	require.NoError(t, err)

	// The integration reference should carry only its ID; the backend does not
	// track an integration name, so an empty `name:` must not be emitted.
	assert.Contains(t, out, "id: "+integrationID)
	assert.NotContains(t, out, "name: \"\"")
	assert.NotContains(t, out, "name: ''")

	// The re-serialized YAML must still round-trip through the parser.
	parsed, err := CanvasFromYAML([]byte(out))
	require.NoError(t, err)
	require.Len(t, parsed.Spec.Nodes, 1)
	require.NotNil(t, parsed.Spec.Nodes[0].Integration)
	assert.Equal(t, integrationID, parsed.Spec.Nodes[0].Integration.ID)
	assert.Equal(t, "", parsed.Spec.Nodes[0].Integration.Name)
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
