package models

import (
	"encoding/json"
	"fmt"

	"github.com/superplanehq/superplane/pkg/openapi_client"
	"gopkg.in/yaml.v3"
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
	var yamlObject any
	if err := yaml.Unmarshal(raw, &yamlObject); err != nil {
		return nil, fmt.Errorf("failed to parse canvas resource: %w", err)
	}

	jsonData, err := json.Marshal(yamlObject)
	if err != nil {
		return nil, fmt.Errorf("failed to convert canvas resource to json: %w", err)
	}

	var resource Canvas
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse canvas json payload: %w", err)
	}

	if resource.Kind != CanvasKind {
		return nil, fmt.Errorf("unsupported resource kind %q", resource.Kind)
	}

	if resource.APIVersion == "" {
		return nil, fmt.Errorf("canvas apiVersion is required")
	}

	if resource.Metadata == nil {
		return nil, fmt.Errorf("canvas metadata is required")
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
