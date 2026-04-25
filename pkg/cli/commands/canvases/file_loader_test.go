package canvases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type testCanvasConfig struct {
	activeCanvas string
}

func (c testCanvasConfig) GetActiveCanvas() string {
	return c.activeCanvas
}

func (c testCanvasConfig) SetActiveCanvas(string) error {
	return nil
}

func TestLoadCanvasForCreateFromFilePreservesPositionAndMetadata(t *testing.T) {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "canvas.yaml")
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: parse-check
spec:
  nodes:
    - id: trigger-1
      name: Trigger
      type: TYPE_TRIGGER
      trigger:
        name: start
      metadata:
        repository:
          id: 123
          name: superplane
      position:
        x: 120
        y: 500
  edges: []
`)
	if err := os.WriteFile(filePath, raw, 0o600); err != nil {
		t.Fatalf("failed to write temp canvas: %v", err)
	}

	canvas, autoLayout, err := loadCanvasForCreateFromFile(filePath)
	if err != nil {
		t.Fatalf("loadCanvasForCreateFromFile returned error: %v", err)
	}
	if autoLayout != nil {
		t.Fatalf("expected autoLayout to be nil when not set in file")
	}

	if canvas.Spec == nil {
		t.Fatalf("expected canvas spec to be set")
	}

	nodes := canvas.Spec.GetNodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	if nodes[0].Position == nil {
		t.Fatalf("expected node position to be set")
	}
	if nodes[0].Position.GetY() != 500 {
		t.Fatalf("expected node y=500, got %d", nodes[0].Position.GetY())
	}
	if nodes[0].GetMetadata() == nil {
		t.Fatalf("expected node metadata to be set")
	}

	repository, ok := nodes[0].GetMetadata()["repository"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata.repository map, got %#v", nodes[0].GetMetadata()["repository"])
	}
	if repository["name"] != "superplane" {
		t.Fatalf("expected metadata.repository.name=superplane, got %#v", repository["name"])
	}
}

func TestLoadCanvasForCreateFromFileParsesAutoLayout(t *testing.T) {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "canvas.yaml")
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: parse-check
spec:
  nodes: []
  edges: []
autoLayout:
  algorithm: ALGORITHM_HORIZONTAL
  scope: SCOPE_FULL_CANVAS
`)
	if err := os.WriteFile(filePath, raw, 0o600); err != nil {
		t.Fatalf("failed to write temp canvas: %v", err)
	}

	_, autoLayout, err := loadCanvasForCreateFromFile(filePath)
	if err != nil {
		t.Fatalf("loadCanvasForCreateFromFile returned error: %v", err)
	}
	if autoLayout == nil {
		t.Fatalf("expected autoLayout to be parsed")
	}
	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected algorithm ALGORITHM_HORIZONTAL, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS {
		t.Fatalf("expected scope SCOPE_FULL_CANVAS, got %s", autoLayout.GetScope())
	}
}

func TestLoadCanvasFromFileRequiresMetadataIDOrActiveCanvas(t *testing.T) {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "canvas.yaml")
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: parse-check
spec:
  nodes: []
  edges: []
`)
	if err := os.WriteFile(filePath, raw, 0o600); err != nil {
		t.Fatalf("failed to write temp canvas: %v", err)
	}

	ctx := core.CommandContext{Config: testCanvasConfig{}}
	_, _, err := loadCanvasFromFile(ctx, filePath)
	if err == nil {
		t.Fatalf("expected validation error when metadata.id and active canvas are unset")
	}
	want := "canvas metadata.id is required in the file when no active canvas is set; set one with `superplane canvases active` or add metadata.id to the YAML"
	if err.Error() != want {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCanvasFromFileUsesActiveCanvasWhenMetadataIDMissing(t *testing.T) {
	t.Helper()

	activeID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	filePath := filepath.Join(t.TempDir(), "canvas.yaml")
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: parse-check
spec:
  nodes: []
  edges: []
`)
	if err := os.WriteFile(filePath, raw, 0o600); err != nil {
		t.Fatalf("failed to write temp canvas: %v", err)
	}

	ctx := core.CommandContext{Config: testCanvasConfig{activeCanvas: activeID}}
	canvasID, canvas, err := loadCanvasFromFile(ctx, filePath)
	if err != nil {
		t.Fatalf("loadCanvasFromFile returned error: %v", err)
	}
	if canvasID != activeID {
		t.Fatalf("expected canvas id %q, got %q", activeID, canvasID)
	}
	if canvas.GetMetadata().GetId() != activeID {
		t.Fatalf("expected metadata id on canvas to be set to %q, got %q", activeID, canvas.GetMetadata().GetId())
	}
}

func TestLoadCanvasFromFileRejectsMetadataIDMismatchWithActiveCanvas(t *testing.T) {
	t.Helper()

	activeID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	fileID := "5f9ae08d-0363-40d2-ba2c-5f6389a418d8"
	filePath := filepath.Join(t.TempDir(), "canvas.yaml")
	raw := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  id: ` + fileID + `
  name: parse-check
spec:
  nodes: []
  edges: []
`)
	if err := os.WriteFile(filePath, raw, 0o600); err != nil {
		t.Fatalf("failed to write temp canvas: %v", err)
	}

	ctx := core.CommandContext{Config: testCanvasConfig{activeCanvas: activeID}}
	_, _, err := loadCanvasFromFile(ctx, filePath)
	if err == nil {
		t.Fatalf("expected id mismatch error")
	}
	want := `canvas metadata.id "` + fileID + `" does not match the active canvas "` + activeID + `"; clear the active canvas or fix metadata.id`
	if err.Error() != want {
		t.Fatalf("unexpected error: %v", err)
	}
}
