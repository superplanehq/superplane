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
        autoCancel: queued
    - id: test
      name: Test
      type: TYPE_ACTION
      component: noop
  edges:
    - sourceId: deploy
      targetId: test
  groups:
    - id: staging-section
      nodes: [deploy, test]
      max: 2
`)

	resource, err := CanvasFromYAML(raw)
	require.NoError(t, err)

	concurrency := resource.Spec.Nodes[0].Concurrency
	require.NotNil(t, concurrency)
	assert.Equal(t, "ci-{{ root().data.branch }}", concurrency.Key)
	require.NotNil(t, concurrency.Max)
	assert.Equal(t, 3, *concurrency.Max)
	assert.Equal(t, "queued", concurrency.AutoCancel)
	assert.Nil(t, resource.Spec.Nodes[1].Concurrency)

	groups := resource.NodeGroups()
	require.Len(t, groups, 1)
	assert.Equal(t, "staging-section", groups[0].ID)
	assert.Equal(t, []string{"deploy", "test"}, groups[0].Nodes)
	assert.Equal(t, 2, groups[0].EffectiveMax())

	nodes := resource.Nodes()
	require.NotNil(t, nodes[0].Concurrency)
	assert.Equal(t, "ci-{{ root().data.branch }}", nodes[0].Concurrency.Key)
}

func TestCanvas_ValidateNodeConcurrency(t *testing.T) {
	limit := func(v int) *int { return &v }
	nodeWithConcurrency := func(concurrency *ConcurrencySpec) Node {
		return Node{ID: "deploy", Concurrency: concurrency}
	}

	t.Run("valid specs pass", func(t *testing.T) {
		assert.NoError(t, validateNodeConcurrency(nodeWithConcurrency(nil)))
		assert.NoError(t, validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Key: "ci-{{ root().data.branch }}"})))
		assert.NoError(t, validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Max: limit(3), AutoCancel: "queued"})))
	})

	t.Run("max must be at least 1", func(t *testing.T) {
		err := validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Max: limit(0)}))
		assert.ErrorContains(t, err, "must be at least 1")

		err = validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{Max: limit(-1)}))
		assert.ErrorContains(t, err, "must be at least 1")
	})

	t.Run("autoCancel must be queued or running", func(t *testing.T) {
		err := validateNodeConcurrency(nodeWithConcurrency(&ConcurrencySpec{AutoCancel: "sometimes"}))
		assert.ErrorContains(t, err, "invalid concurrency autoCancel")
	})
}

func TestCanvas_ValidateGroups(t *testing.T) {
	limit := func(v int) *int { return &v }
	nodeIDs := map[string]bool{"deploy": true, "test": true, "notify": true}
	canvasWithGroups := func(groups ...NodeGroup) *Canvas {
		return &Canvas{Spec: &CanvasSpec{Groups: groups}}
	}

	t.Run("valid groups pass", func(t *testing.T) {
		err := canvasWithGroups(
			NodeGroup{ID: "staging", Nodes: []string{"deploy", "test"}, Max: limit(2)},
			NodeGroup{ID: "post", Nodes: []string{"notify"}},
		).validateGroups(nodeIDs)
		assert.NoError(t, err)
	})

	t.Run("id is required", func(t *testing.T) {
		err := canvasWithGroups(NodeGroup{Nodes: []string{"deploy"}}).validateGroups(nodeIDs)
		assert.ErrorContains(t, err, "id is required")
	})

	t.Run("duplicate group ids are rejected", func(t *testing.T) {
		err := canvasWithGroups(
			NodeGroup{ID: "staging", Nodes: []string{"deploy"}},
			NodeGroup{ID: "staging", Nodes: []string{"test"}},
		).validateGroups(nodeIDs)
		assert.ErrorContains(t, err, "duplicate group id")
	})

	t.Run("max must be at least 1", func(t *testing.T) {
		err := canvasWithGroups(NodeGroup{ID: "staging", Nodes: []string{"deploy"}, Max: limit(0)}).validateGroups(nodeIDs)
		assert.ErrorContains(t, err, "must be at least 1")
	})

	t.Run("nodes are required", func(t *testing.T) {
		err := canvasWithGroups(NodeGroup{ID: "staging"}).validateGroups(nodeIDs)
		assert.ErrorContains(t, err, "nodes is required")
	})

	t.Run("unknown nodes are rejected", func(t *testing.T) {
		err := canvasWithGroups(NodeGroup{ID: "staging", Nodes: []string{"missing"}}).validateGroups(nodeIDs)
		assert.ErrorContains(t, err, "node missing not found")
	})

	t.Run("groups must be disjoint", func(t *testing.T) {
		err := canvasWithGroups(
			NodeGroup{ID: "staging", Nodes: []string{"deploy"}},
			NodeGroup{ID: "other", Nodes: []string{"deploy"}},
		).validateGroups(nodeIDs)
		assert.ErrorContains(t, err, "more than one group")
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
					Key:        "ci-{{ root().data.branch }}",
					Max:        limit(3),
					AutoCancel: "queued",
				},
			},
		}),
		Edges: datatypes.NewJSONSlice([]models.Edge{}),
		NodeGroups: datatypes.NewJSONSlice([]models.NodeGroup{
			{ID: "staging-section", Nodes: []string{"deploy"}, Max: limit(2)},
		}),
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
	assert.Equal(t, "queued", concurrency.AutoCancel)

	require.Len(t, roundTripped.Spec.Groups, 1)
	assert.Equal(t, NodeGroup{ID: "staging-section", Nodes: []string{"deploy"}, Max: limit(2)}, roundTripped.Spec.Groups[0])
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
