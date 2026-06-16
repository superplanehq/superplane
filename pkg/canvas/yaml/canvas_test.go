package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCanvasResourceRejectsInvalidKind(t *testing.T) {
	_, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Workflow
metadata:
  name: test
spec:
  nodes: []
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource kind")
}

func TestParseCanvasResourceRequiresMetadata(t *testing.T) {
	_, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Canvas
spec:
  nodes: []
`))
	require.Error(t, err)
}

func TestParseCanvasResourceRejectsTemplateMetadata(t *testing.T) {
	_, err := ParseCanvasResource([]byte(`apiVersion: v1
kind: Canvas
metadata:
  id: canvas-id
  name: test
  isTemplate: true
spec:
  nodes: []
  edges: []
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
	assert.Contains(t, err.Error(), "isTemplate")
}
