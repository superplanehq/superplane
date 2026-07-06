package yaml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	APIVersion  = "v1"
	KindCanvas  = "Canvas"
	KindConsole = "Console"
)

type Canvas struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Kind       string         `json:"kind" yaml:"kind"`
	Metadata   CanvasMetadata `json:"metadata" yaml:"metadata"`
	Spec       CanvasSpec     `json:"spec" yaml:"spec"`
}

type CanvasMetadata struct {
	CanvasID string `json:"canvasId,omitempty" yaml:"canvasId,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
}

type CanvasSpec struct {
	Nodes []CanvasNode `json:"nodes" yaml:"nodes"`
	Edges []CanvasEdge `json:"edges" yaml:"edges"`
}

type CanvasNode struct {
	ID            string             `json:"id" yaml:"id"`
	Name          string             `json:"name" yaml:"name"`
	Type          string             `json:"type" yaml:"type"`
	Component     string             `json:"component" yaml:"component"`
	Configuration map[string]any     `json:"configuration" yaml:"configuration"`
	Metadata      map[string]any     `json:"metadata" yaml:"metadata"`
	Position      CanvasNodePosition `json:"position" yaml:"position"`
	IsCollapsed   bool               `json:"isCollapsed" yaml:"isCollapsed"`
	IntegrationID *string            `json:"integrationId,omitempty" yaml:"integrationId,omitempty"`
}

type CanvasNodePosition struct {
	X int `json:"x" yaml:"x"`
	Y int `json:"y" yaml:"y"`
}

type CanvasEdge struct {
	SourceID string `json:"sourceId" yaml:"sourceId"`
	TargetID string `json:"targetId" yaml:"targetId"`
	Channel  string `json:"channel" yaml:"channel"`
}

func CanvasVersionToCanvasYAML(canvasName string, canvasVersion *models.CanvasVersion) ([]byte, error) {
	if canvasVersion == nil {
		return nil, errors.New("canvas version is required")
	}

	resource := Canvas{
		APIVersion: APIVersion,
		Kind:       KindCanvas,
		Metadata: CanvasMetadata{
			Name: canvasName,
		},
		Spec: CanvasSpec{
			Nodes: []CanvasNode{},
			Edges: []CanvasEdge{},
		},
	}

	for _, edge := range canvasVersion.Edges {
		resource.Spec.Edges = append(resource.Spec.Edges, CanvasEdge{
			SourceID: edge.SourceID,
			TargetID: edge.TargetID,
			Channel:  edge.Channel,
		})
	}

	for _, node := range canvasVersion.Nodes {
		resource.Spec.Nodes = append(resource.Spec.Nodes, CanvasNode{
			ID:            node.ID,
			Name:          node.Name,
			Type:          node.Type,
			Component:     node.ComponentName(),
			Configuration: node.Configuration,
			Metadata:      node.Metadata,
			Position:      CanvasNodePosition{X: node.Position.X, Y: node.Position.Y},
			IsCollapsed:   node.IsCollapsed,
			IntegrationID: node.IntegrationID,
		})
	}

	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize dashboard: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("failed to serialize dashboard: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("failed to encode dashboard yaml: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to encode dashboard yaml: %w", err)
	}

	return buf.Bytes(), nil
}
