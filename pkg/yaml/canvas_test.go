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

func TestCanvasFromYAML_ReadsQueueConfig(t *testing.T) {
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
      queue:
        key: "ci-{{ root().data.branch }}"
        maxParallelism: 3
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
      maxParallelism: 2
`)

	resource, err := CanvasFromYAML(raw)
	require.NoError(t, err)

	queue := resource.Spec.Nodes[0].Queue
	require.NotNil(t, queue)
	assert.Equal(t, "ci-{{ root().data.branch }}", queue.Key)
	require.NotNil(t, queue.MaxParallelism)
	assert.Equal(t, 3, *queue.MaxParallelism)
	assert.Equal(t, "queued", queue.AutoCancel)
	assert.Nil(t, resource.Spec.Nodes[1].Queue)

	groups := resource.NodeGroups()
	require.Len(t, groups, 1)
	assert.Equal(t, "staging-section", groups[0].ID)
	assert.Equal(t, []string{"deploy", "test"}, groups[0].Nodes)
	assert.Equal(t, 2, groups[0].EffectiveMaxParallelism())

	nodes := resource.Nodes()
	require.NotNil(t, nodes[0].Queue)
	assert.Equal(t, "ci-{{ root().data.branch }}", nodes[0].Queue.Key)
}

func TestCanvas_ValidateNodeQueue(t *testing.T) {
	limit := func(v int) *int { return &v }
	nodeWithQueue := func(queue *QueueSpec) Node {
		return Node{ID: "deploy", Queue: queue}
	}

	t.Run("valid specs pass", func(t *testing.T) {
		assert.NoError(t, validateNodeQueue(nodeWithQueue(nil)))
		assert.NoError(t, validateNodeQueue(nodeWithQueue(&QueueSpec{Key: "ci-{{ root().data.branch }}"})))
		assert.NoError(t, validateNodeQueue(nodeWithQueue(&QueueSpec{MaxParallelism: limit(3), AutoCancel: "queued"})))
		assert.NoError(t, validateNodeQueue(nodeWithQueue(&QueueSpec{MaxParallelism: limit(0)})))
	})

	t.Run("maxParallelism must not be negative", func(t *testing.T) {
		err := validateNodeQueue(nodeWithQueue(&QueueSpec{MaxParallelism: limit(-1)}))
		assert.ErrorContains(t, err, "must not be negative")
	})

	t.Run("autoCancel must be queued or running", func(t *testing.T) {
		err := validateNodeQueue(nodeWithQueue(&QueueSpec{AutoCancel: "sometimes"}))
		assert.ErrorContains(t, err, "invalid queue autoCancel")
	})

	t.Run("autoCancel with unlimited parallelism is rejected", func(t *testing.T) {
		err := validateNodeQueue(nodeWithQueue(&QueueSpec{MaxParallelism: limit(0), AutoCancel: "queued"}))
		assert.ErrorContains(t, err, "no effect with maxParallelism 0")
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
			NodeGroup{ID: "staging", Nodes: []string{"deploy", "test"}, MaxParallelism: limit(2)},
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

	t.Run("maxParallelism must be at least 1", func(t *testing.T) {
		err := canvasWithGroups(NodeGroup{ID: "staging", Nodes: []string{"deploy"}, MaxParallelism: limit(0)}).validateGroups(nodeIDs)
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

func TestVersionToCanvasYAML_IncludesQueueConfig(t *testing.T) {
	limit := func(v int) *int { return &v }
	version := &models.CanvasVersion{
		Nodes: datatypes.NewJSONSlice([]models.Node{
			{
				ID:   "deploy",
				Name: "Deploy",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
				Queue: &models.QueueSpec{
					Key:            "ci-{{ root().data.branch }}",
					MaxParallelism: limit(3),
					AutoCancel:     "queued",
				},
			},
		}),
		Edges: datatypes.NewJSONSlice([]models.Edge{}),
		NodeGroups: datatypes.NewJSONSlice([]models.NodeGroup{
			{ID: "staging-section", Nodes: []string{"deploy"}, MaxParallelism: limit(2)},
		}),
	}

	out, err := VersionToCanvasYAML("test", "", version)
	require.NoError(t, err)

	roundTripped, err := CanvasFromYAML([]byte(out))
	require.NoError(t, err)

	queue := roundTripped.Spec.Nodes[0].Queue
	require.NotNil(t, queue)
	assert.Equal(t, "ci-{{ root().data.branch }}", queue.Key)
	require.NotNil(t, queue.MaxParallelism)
	assert.Equal(t, 3, *queue.MaxParallelism)
	assert.Equal(t, "queued", queue.AutoCancel)

	require.Len(t, roundTripped.Spec.Groups, 1)
	assert.Equal(t, NodeGroup{ID: "staging-section", Nodes: []string{"deploy"}, MaxParallelism: limit(2)}, roundTripped.Spec.Groups[0])
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
