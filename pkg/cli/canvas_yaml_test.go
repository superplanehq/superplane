package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__ParseCanvasYaml__ValidCanvas(t *testing.T) {
	resource := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: test-canvas
  description: A test canvas
spec:
  nodes: []
  edges: []`)

	apiVersion, kind, err := ParseYamlResourceHeaders(resource)

	assert.Nil(t, err)
	assert.Equal(t, "v1", apiVersion)
	assert.Equal(t, "Canvas", kind)
}

func Test__ParseCanvasYaml__CanvasWithNodes(t *testing.T) {
	resource := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: test-canvas-with-nodes
spec:
  nodes:
    - id: node1
      name: Test Node
      type: TYPE_COMPONENT
      component:
        name: noop
  edges: []`)

	apiVersion, kind, err := ParseYamlResourceHeaders(resource)

	assert.Nil(t, err)
	assert.Equal(t, "v1", apiVersion)
	assert.Equal(t, "Canvas", kind)
}

func Test__ParseCanvasYaml__MinimalCanvas(t *testing.T) {
	resource := []byte(`
apiVersion: v1
kind: Canvas
metadata:
  name: minimal-canvas`)

	apiVersion, kind, err := ParseYamlResourceHeaders(resource)

	assert.Nil(t, err)
	assert.Equal(t, "v1", apiVersion)
	assert.Equal(t, "Canvas", kind)
}
