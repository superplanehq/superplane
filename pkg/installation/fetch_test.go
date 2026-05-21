package installation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCanvasYAMLWithResourceHeaders(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Preview Environments
  description: StoreJS preview
  isTemplate: false
spec:
  nodes: []
  edges: []
`)

	canvas, err := parseCanvasYAML(raw)
	require.NoError(t, err)
	assert.Equal(t, "Preview Environments", canvas.GetMetadata().GetName())
	assert.Equal(t, "StoreJS preview", canvas.GetMetadata().GetDescription())
	assert.False(t, canvas.GetMetadata().GetIsTemplate())
	assert.Empty(t, canvas.GetMetadata().GetId())
}

func TestParseCanvasYAMLRejectsTemplate(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Template App
  isTemplate: true
spec:
  nodes: []
  edges: []
`)

	_, err := parseCanvasYAML(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template")
}

func TestParseCanvasYAMLRejectsUnsupportedKind(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Dashboard
metadata:
  name: Not a canvas
spec:
  panels: []
`)

	_, err := parseCanvasYAML(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported resource kind "Dashboard"`)
}

func TestFetchCanvasFromPublicRepo(t *testing.T) {
	repo, err := ParseRepository("github.com/superplanehq/preview-env-github-digitalocean")
	require.NoError(t, err)

	canvas, ref, err := FetchCanvas(repo)
	require.NoError(t, err)
	assert.Equal(t, "main", ref)
	assert.NotEmpty(t, canvas.GetMetadata().GetName())
	assert.NotEmpty(t, canvas.GetSpec().GetNodes())
}
