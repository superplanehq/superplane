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
	//
	// Using the previously defined API types
	// for backwards compatibility.
	//
	NodeTypeTrigger = "TYPE_TRIGGER"
	NodeTypeWidget  = "TYPE_WIDGET"
	NodeTypeAction  = "TYPE_ACTION"
)

type Canvas struct {
	APIVersion string          `json:"apiVersion" yaml:"apiVersion"`
	Kind       string          `json:"kind" yaml:"kind"`
	Metadata   *CanvasMetadata `json:"metadata" yaml:"metadata"`
	Spec       *CanvasSpec     `json:"spec" yaml:"spec"`
}

func (c *Canvas) Nodes() []models.Node {
	nodes := make([]models.Node, len(c.Spec.Nodes))
	for i, node := range c.Spec.Nodes {
		nodes[i] = node.Model()
	}
	return nodes
}

func (c *Canvas) Edges() []models.Edge {
	edges := make([]models.Edge, len(c.Spec.Edges))
	for i, edge := range c.Spec.Edges {
		edges[i] = edge.Model()
	}
	return edges
}

type CanvasMetadata struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

type CanvasSpec struct {
	Nodes []Node `json:"nodes" yaml:"nodes"`
	Edges []Edge `json:"edges" yaml:"edges"`
}

type Edge struct {
	SourceID string `json:"sourceId" yaml:"sourceId"`
	TargetID string `json:"targetId" yaml:"targetId"`
	Channel  string `json:"channel" yaml:"channel"`
}

func (e *Edge) Model() models.Edge {
	return models.Edge{
		SourceID: e.SourceID,
		TargetID: e.TargetID,
		Channel:  e.Channel,
	}
}

type Node struct {
	ID             string          `json:"id" yaml:"id"`
	Name           string          `json:"name" yaml:"name"`
	Type           string          `json:"type" yaml:"type"`
	Component      string          `json:"component" yaml:"component"`
	Configuration  map[string]any  `json:"configuration" yaml:"configuration"`
	Position       Position        `json:"position" yaml:"position"`
	IsCollapsed    bool            `json:"isCollapsed" yaml:"isCollapsed"`
	Metadata       map[string]any  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Integration    *IntegrationRef `json:"integration,omitempty" yaml:"integration,omitempty"`
	ErrorMessage   *string         `json:"errorMessage,omitempty" yaml:"errorMessage,omitempty"`
	WarningMessage *string         `json:"warningMessage,omitempty" yaml:"warningMessage,omitempty"`
}

type IntegrationRef struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

func (n *Node) NodeTypeForModel() string {
	switch n.Type {
	case NodeTypeTrigger:
		return models.NodeTypeTrigger
	case NodeTypeWidget:
		return models.NodeTypeWidget
	case NodeTypeAction:
		return models.NodeTypeComponent
	default:
		return ""
	}
}

func (n *Node) Model() models.Node {
	model := models.Node{
		ID:             n.ID,
		Name:           n.Name,
		Type:           n.NodeTypeForModel(),
		Configuration:  n.Configuration,
		Metadata:       n.Metadata,
		IsCollapsed:    n.IsCollapsed,
		ErrorMessage:   n.ErrorMessage,
		WarningMessage: n.WarningMessage,
		Position: models.Position{
			X: n.Position.X,
			Y: n.Position.Y,
		},
	}

	if n.Integration != nil {
		model.IntegrationID = &n.Integration.ID
	}

	if n.Type == NodeTypeAction {
		model.Ref = models.NodeRef{
			Component: &models.ComponentRef{
				Name: n.Component,
			},
		}
	}

	if n.Type == NodeTypeTrigger {
		model.Ref = models.NodeRef{
			Trigger: &models.TriggerRef{
				Name: n.Component,
			},
		}
	}

	if n.Type == NodeTypeWidget {
		model.Ref = models.NodeRef{
			Widget: &models.WidgetRef{
				Name: n.Component,
			},
		}
	}

	return model
}

type Position struct {
	X int `json:"x" yaml:"x"`
	Y int `json:"y" yaml:"y"`
}

func CanvasFromYAML(raw []byte) (*Canvas, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("canvas yaml is empty")
	}

	var asAny any
	if err := yaml.Unmarshal(raw, &asAny); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	if _, ok := asAny.(map[string]any); !ok {
		return nil, errors.New("canvas yaml must be an object")
	}

	jsonBytes, err := json.Marshal(asAny)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()

	var resource Canvas
	if err := decoder.Decode(&resource); err != nil {
		return nil, fmt.Errorf("invalid canvas yaml: %w", err)
	}

	// TODO: put validation from ParseCanvas() here
	// if err := resource.Validate(); err != nil {
	// 	return nil, err
	// }

	return &resource, nil
}

func VersionToCanvasYAML(name string, description string, canvasVersion *models.CanvasVersion) (string, error) {
	if canvasVersion == nil {
		return "", errors.New("canvas version is required")
	}

	//
	// Some sort of stable ordering of nodes and edges would be nice.
	//
	resource := Canvas{
		APIVersion: APIVersion,
		Kind:       KindCanvas,
		Metadata: &CanvasMetadata{
			Name:        name,
			Description: description,
		},
		Spec: &CanvasSpec{
			Nodes: []Node{},
			Edges: []Edge{},
		},
	}

	for _, node := range canvasVersion.Nodes {
		n := Node{
			ID:             node.ID,
			Name:           node.Name,
			Type:           ModelTypeToYamlType(node.Type),
			Component:      NodeComponentName(node.Ref),
			Configuration:  node.Configuration,
			Metadata:       node.Metadata,
			IsCollapsed:    node.IsCollapsed,
			ErrorMessage:   node.ErrorMessage,
			WarningMessage: node.WarningMessage,
			Position: Position{
				X: node.Position.X,
				Y: node.Position.Y,
			},
		}

		if node.IntegrationID != nil {
			n.Integration = &IntegrationRef{
				ID: *node.IntegrationID,
			}
		}

		resource.Spec.Nodes = append(resource.Spec.Nodes, n)
	}

	for _, edge := range canvasVersion.Edges {
		resource.Spec.Edges = append(resource.Spec.Edges, Edge{
			SourceID: edge.SourceID,
			TargetID: edge.TargetID,
			Channel:  edge.Channel,
		})
	}

	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return "", fmt.Errorf("failed to serialize canvas: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return "", fmt.Errorf("failed to serialize canvas: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return "", fmt.Errorf("failed to encode canvas yaml: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("failed to encode canvas yaml: %w", err)
	}

	return buf.String(), nil
}

func ModelTypeToYamlType(t string) string {
	switch t {
	case models.NodeTypeTrigger:
		return NodeTypeTrigger
	case models.NodeTypeWidget:
		return NodeTypeWidget
	case models.NodeTypeComponent:
		return NodeTypeAction
	default:
		return ""
	}
}

func NodeComponentName(ref models.NodeRef) string {
	if ref.Component != nil && ref.Component.Name != "" {
		return ref.Component.Name
	}

	if ref.Trigger != nil && ref.Trigger.Name != "" {
		return ref.Trigger.Name
	}

	if ref.Widget != nil && ref.Widget.Name != "" {
		return ref.Widget.Name
	}

	return ""
}
