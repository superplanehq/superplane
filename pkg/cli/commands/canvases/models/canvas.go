package models

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	CanvasKind = "Canvas"
)

type Canvas struct {
	APIVersion string                                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                                 `json:"kind" yaml:"kind"`
	Metadata   *openapi_client.CanvasesCanvasMetadata `json:"metadata" yaml:"metadata"`
	Spec       *openapi_client.CanvasesCanvasSpec     `json:"spec,omitempty" yaml:"spec,omitempty"`
}

func ParseCanvas(raw []byte) (*Canvas, error) {
	var resource Canvas
	if err := yaml.Unmarshal(raw, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse canvas resource: %w", err)
	}

	if resource.Kind != CanvasKind {
		return nil, fmt.Errorf("unsupported resource kind %q", resource.Kind)
	}

	if resource.APIVersion == "" {
		return nil, fmt.Errorf("canvas apiVersion is required")
	}

	if resource.Metadata.Name == nil {
		return nil, fmt.Errorf("canvas metadata.name is required")
	}

	return &resource, nil
}

func CanvasFromCanvas(resource Canvas) openapi_client.CanvasesCanvas {
	canvas := openapi_client.CanvasesCanvas{}
	canvas.SetMetadata(*resource.Metadata)
	canvas.SetSpec(*resource.Spec)
	return canvas
}

func CanvasResourceFromCanvas(canvas openapi_client.CanvasesCanvas) Canvas {
	return Canvas{
		APIVersion: "v1",
		Kind:       CanvasKind,
		Metadata:   canvas.Metadata,
		Spec:       canvas.Spec,
	}
}

func EmptyCanvasSpec() *openapi_client.CanvasesCanvasSpec {
	return &openapi_client.CanvasesCanvasSpec{
		Nodes: []openapi_client.ComponentsNode{},
		Edges: []openapi_client.ComponentsEdge{},
	}
}
