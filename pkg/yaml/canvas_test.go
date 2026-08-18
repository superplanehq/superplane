package yaml

import (
	"bytes"
	"encoding/json"
	"testing"

	ghodssyaml "github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
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

func TestCanvasFromYAML_ReadsConcurrencyConfig(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: test
spec:
  nodes:
    - id: deploy
      name: Deploy
      type: TYPE_ACTION
      component: noop
      concurrency:
        key: "ci-{{ root().data.branch }}"
        max: 3
    - id: test
      name: Test
      type: TYPE_ACTION
      component: noop
  edges:
    - sourceId: deploy
      targetId: test
`)

	resource, err := CanvasFromYAML(raw)
	require.NoError(t, err)

	concurrency := resource.Spec.Nodes[0].Concurrency
	require.NotNil(t, concurrency)
	assert.Equal(t, "ci-{{ root().data.branch }}", concurrency.Key)
	require.NotNil(t, concurrency.Max)
	assert.Equal(t, 3, *concurrency.Max)
	assert.Nil(t, resource.Spec.Nodes[1].Concurrency)

	nodes := resource.Nodes()
	require.NotNil(t, nodes[0].Concurrency)
	assert.Equal(t, "ci-{{ root().data.branch }}", nodes[0].Concurrency.Key)
}

func TestCanvas_ValidateNodeConcurrency(t *testing.T) {
	limit := func(v int) *int { return &v }
	nodeWithConcurrency := func(concurrency *ConcurrencySpec) Node {
		return Node{ID: "deploy", Type: NodeTypeAction, Component: "http", Concurrency: concurrency}
	}

	t.Run("valid specs pass", func(t *testing.T) {
		assert.NoError(t, validateNodeConcurrency(nodeWithConcurrency(nil)))
		assert.NoError(t, validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Key: "ci-{{ root().data.branch }}"})))
		assert.NoError(t, validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Max: limit(3)})))
	})

	t.Run("max must be at least 1", func(t *testing.T) {
		err := validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Max: limit(0)}))
		assert.ErrorContains(t, err, "must be at least 1")

		err = validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Max: limit(-1)}))
		assert.ErrorContains(t, err, "must be at least 1")
	})

	t.Run("only action nodes take a spec", func(t *testing.T) {
		node := Node{ID: "on-push", Type: NodeTypeTrigger, Component: "github.onPush", Concurrency: &ConcurrencySpec{Max: limit(2)}}
		err := validateNodeConcurrency(node)
		assert.ErrorContains(t, err, "only supported on action nodes")
	})

	t.Run("merge does not support concurrency", func(t *testing.T) {
		node := Node{ID: "merge", Type: NodeTypeAction, Component: "merge", Concurrency: &ConcurrencySpec{Max: limit(2)}}
		err := validateNodeConcurrency(node)
		assert.ErrorContains(t, err, "merge component does not support concurrency")

		node.Concurrency = nil
		assert.NoError(t, validateNodeConcurrency(node))
	})

	t.Run("loop supports max only", func(t *testing.T) {
		node := Node{ID: "loop", Type: NodeTypeAction, Component: "loop", Concurrency: &ConcurrencySpec{Max: limit(2)}}
		assert.NoError(t, validateNodeConcurrency(node))

		node.Concurrency = &ConcurrencySpec{Key: "{{ root().data.branch }}"}
		err := validateNodeConcurrency(node)
		assert.ErrorContains(t, err, "loop component does not support concurrency key")
	})
}

func TestVersionToCanvasYAML_IncludesConcurrencyConfig(t *testing.T) {
	limit := func(v int) *int { return &v }
	version := &models.CanvasVersion{
		Nodes: datatypes.NewJSONSlice([]models.Node{
			{
				ID:   "deploy",
				Name: "Deploy",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
				Concurrency: &models.ConcurrencySpec{
					Key: "ci-{{ root().data.branch }}",
					Max: limit(3),
				},
			},
		}),
		Edges: datatypes.NewJSONSlice([]models.Edge{}),
	}

	out, err := VersionToCanvasYAML("test", "", version)
	require.NoError(t, err)

	roundTripped, err := CanvasFromYAML([]byte(out))
	require.NoError(t, err)

	concurrency := roundTripped.Spec.Nodes[0].Concurrency
	require.NotNil(t, concurrency)
	assert.Equal(t, "ci-{{ root().data.branch }}", concurrency.Key)
	require.NotNil(t, concurrency.Max)
	assert.Equal(t, 3, *concurrency.Max)
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
