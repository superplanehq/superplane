package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestParseCanvasPreservesPositionYFromUnquotedKey(t *testing.T) {
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  id: 4e9ae08d-0363-40d2-ba2c-5f6389a418d8
  name: advanced-scala-issue-plan-discord
spec:
  nodes:
    - id: manual-plan-start
      name: manual_plan_start
      type: TYPE_TRIGGER
      component: start
      configuration:
        templates:
          - name: Incident Report
            payload:
              incidentId: INC-1001
      position:
        x: 120
        y: 500
      paused: false
      isCollapsed: false
  edges:
    - sourceId: manual-plan-start
      targetId: manual-plan-start
      channel: default
`)

	resource, err := ParseCanvas(raw)
	if err != nil {
		t.Fatalf("ParseCanvas returned error: %v", err)
	}

	nodes := resource.Spec.GetNodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	position := nodes[0].GetPosition()
	if position.GetX() != 120 {
		t.Fatalf("expected x=120, got %d", position.GetX())
	}
	if position.GetY() != 500 {
		t.Fatalf("expected y=500, got %d", position.GetY())
	}

	edges := resource.Spec.GetEdges()
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].GetSourceId() != "manual-plan-start" {
		t.Fatalf("expected sourceId=manual-plan-start, got %q", edges[0].GetSourceId())
	}
	if edges[0].GetTargetId() != "manual-plan-start" {
		t.Fatalf("expected targetId=manual-plan-start, got %q", edges[0].GetTargetId())
	}
}

func TestParseCanvasRejectsUnknownNodeComponentFields(t *testing.T) {
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: unknown-field-test
spec:
  changeManagement:
    enabled: false
  edges: []
  nodes:
    - id: wait-1
      name: wait
      type: TYPE_ACTION
      component: wait
      hello: what
`)

	_, err := ParseCanvas(raw)
	if err == nil {
		t.Fatalf("expected ParseCanvas to fail for unknown field")
	}

	assert.ErrorContains(t, err, "failed to parse canvas yaml")
	assert.ErrorContains(t, err, `unknown field "hello"`)
}

func TestParseCanvasValidationErrors(t *testing.T) {
	t.Run("rejects unsupported kind", func(t *testing.T) {
		raw := []byte(`
apiVersion: v1
kind: NotCanvas
metadata:
  name: invalid-kind
spec:
  nodes: []
  edges: []
`)

		_, err := ParseCanvas(raw)
		assert.EqualError(t, err, `unsupported resource kind "NotCanvas"`)
	})

	t.Run("rejects missing apiVersion", func(t *testing.T) {
		raw := []byte(`
kind: Canvas
metadata:
  name: missing-version
spec:
  nodes: []
  edges: []
`)

		_, err := ParseCanvas(raw)
		assert.EqualError(t, err, "canvas apiVersion is required")
	})

	t.Run("rejects missing metadata", func(t *testing.T) {
		raw := []byte(`
apiVersion: v1
kind: Canvas
spec:
  nodes: []
  edges: []
`)

		_, err := ParseCanvas(raw)
		assert.EqualError(t, err, "canvas metadata is required")
	})

	t.Run("rejects missing metadata.name", func(t *testing.T) {
		raw := []byte(`
apiVersion: v1
kind: Canvas
metadata: {}
spec:
  nodes: []
  edges: []
`)

		_, err := ParseCanvas(raw)
		assert.EqualError(t, err, "canvas metadata.name is required")
	})

	t.Run("rejects invalid yaml", func(t *testing.T) {
		raw := []byte("apiVersion: v1\nkind: Canvas\nmetadata: [\n")
		_, err := ParseCanvas(raw)
		assert.ErrorContains(t, err, "failed to parse canvas yaml")
		assert.ErrorContains(t, err, "invalid yaml")
	})
}

func TestCanvasConversions(t *testing.T) {
	metadata := openapi_client.NewCanvasesCanvasMetadata()
	metadata.SetName("my-canvas")
	metadata.SetId("canvas-id")

	spec := EmptyCanvasSpec()

	resource := Canvas{
		APIVersion: "v1",
		Kind:       CanvasKind,
		Metadata:   metadata,
		Spec:       spec,
	}

	t.Run("CanvasFromCanvas", func(t *testing.T) {
		canvas := CanvasFromCanvas(resource)
		metadata := canvas.GetMetadata()
		spec := canvas.GetSpec()
		assert.Equal(t, "my-canvas", metadata.GetName())
		assert.Equal(t, "canvas-id", metadata.GetId())
		assert.NotNil(t, spec.GetNodes())
		assert.NotNil(t, spec.GetEdges())
	})

	t.Run("CanvasResourceFromCanvas", func(t *testing.T) {
		canvas := CanvasFromCanvas(resource)
		resourceFromCanvas := CanvasResourceFromCanvas(canvas)

		assert.Equal(t, "v1", resourceFromCanvas.APIVersion)
		assert.Equal(t, CanvasKind, resourceFromCanvas.Kind)
		assert.Equal(t, "my-canvas", resourceFromCanvas.Metadata.GetName())
		assert.NotNil(t, resourceFromCanvas.Spec.GetNodes())
		assert.NotNil(t, resourceFromCanvas.Spec.GetEdges())
	})
}

func TestCreateCanvasRequestFromCanvas(t *testing.T) {
	metadata := openapi_client.NewCanvasesCanvasMetadata()
	metadata.SetName("my-canvas")

	t.Run("without autoLayout", func(t *testing.T) {
		resource := Canvas{
			APIVersion: "v1",
			Kind:       CanvasKind,
			Metadata:   metadata,
			Spec:       EmptyCanvasSpec(),
		}

		request := CreateCanvasRequestFromCanvas(resource)
		assert.True(t, request.HasCanvas())
		assert.False(t, request.HasAutoLayout())
		canvas := request.GetCanvas()
		metadata := canvas.GetMetadata()
		assert.Equal(t, "my-canvas", metadata.GetName())
	})

	t.Run("with autoLayout", func(t *testing.T) {
		autoLayout := openapi_client.NewCanvasesCanvasAutoLayout()
		autoLayout.SetNodeIds([]string{"node-1", "node-2"})

		resource := Canvas{
			APIVersion: "v1",
			Kind:       CanvasKind,
			Metadata:   metadata,
			Spec:       EmptyCanvasSpec(),
			AutoLayout: autoLayout,
		}

		request := CreateCanvasRequestFromCanvas(resource)
		assert.True(t, request.HasCanvas())
		assert.True(t, request.HasAutoLayout())
		requestAutoLayout := request.GetAutoLayout()
		assert.Equal(t, []string{"node-1", "node-2"}, requestAutoLayout.GetNodeIds())
	})
}

func TestEmptyCanvasSpec(t *testing.T) {
	spec := EmptyCanvasSpec()
	assert.NotNil(t, spec)
	assert.NotNil(t, spec.Nodes)
	assert.NotNil(t, spec.Edges)
	assert.Len(t, spec.Nodes, 0)
	assert.Len(t, spec.Edges, 0)
}
