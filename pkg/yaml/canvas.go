package yaml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
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
	doc, ok := asAny.(map[string]any)
	if !ok {
		return nil, errors.New("canvas yaml must be an object")
	}

	normalizeCanvasDocument(doc)

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()

	var resource Canvas
	if err := decoder.Decode(&resource); err != nil {
		return nil, fmt.Errorf("invalid canvas yaml: %w", err)
	}

	if resource.APIVersion == "" {
		return nil, errors.New("canvas yaml must include an apiVersion")
	}

	if resource.APIVersion != APIVersion {
		return nil, fmt.Errorf("unsupported apiVersion %q (expected %q)", resource.APIVersion, APIVersion)
	}

	if resource.Kind == "" {
		return nil, errors.New("canvas yaml must include a kind")
	}

	if resource.Kind != KindCanvas {
		return nil, errors.New("canvas yaml must include a kind of canvas")
	}

	if resource.Spec == nil {
		return nil, errors.New("canvas yaml must include a spec block")
	}

	if resource.Metadata == nil {
		return nil, errors.New("canvas yaml must include a metadata block")
	}

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
			Component:      node.ComponentName(),
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

func (c *Canvas) Parse(registry *registry.Registry, orgID string) ([]models.Node, []models.Edge, error) {

	//
	// Allow empty canvases
	//
	if len(c.Spec.Nodes) == 0 {
		if len(c.Spec.Edges) > 0 {
			return nil, nil, fmt.Errorf("canvas has edges but no nodes")
		}
		return []models.Node{}, []models.Edge{}, nil
	}

	nodeIDs := make(map[string]bool)
	nodeTypeByID := make(map[string]string)
	nodeValidationErrors := make(map[string]string)

	for i, node := range c.Spec.Nodes {
		if node.ID == "" {
			return nil, nil, fmt.Errorf("node %d: id is required", i)
		}

		if node.Name == "" {
			return nil, nil, fmt.Errorf("node %s: name is required", node.ID)
		}

		if nodeIDs[node.ID] {
			return nil, nil, fmt.Errorf("node %s: duplicate node id", node.ID)
		}

		if node.Type == "" {
			return nil, nil, fmt.Errorf("node %s: type is required", node.ID)
		}

		if node.Type != NodeTypeTrigger && node.Type != NodeTypeWidget && node.Type != NodeTypeAction {
			return nil, nil, fmt.Errorf("node %s: invalid type %q", node.ID, node.Type)
		}

		nodeIDs[node.ID] = true
		nodeTypeByID[node.ID] = node.Type
		if err := c.validateNodeRef(registry, orgID, node); err != nil {
			nodeValidationErrors[node.ID] = err.Error()
		}
	}

	// Find shadowed names within connected components
	nodeWarnings := c.FindShadowedNameWarnings(registry)

	nodesByID := make(map[string]models.Node, len(c.Spec.Nodes))
	for _, node := range c.Spec.Nodes {
		nodesByID[node.ID] = node.Model()
	}

	//
	// Validate expressions
	//
	for nodeID, errs := range expressionvalidation.ValidateCanvasExpressions(registry, c.Nodes()) {
		msgs := make([]string, 0, len(errs))
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		joined := strings.Join(msgs, "\n")
		if existing, ok := nodeValidationErrors[nodeID]; ok {
			nodeValidationErrors[nodeID] = existing + "\n" + joined
		} else {
			nodeValidationErrors[nodeID] = joined
		}
	}

	//
	// Validate edges
	//
	for i, edge := range c.Spec.Edges {
		if edge.SourceID == "" || edge.TargetID == "" {
			return nil, nil, fmt.Errorf("edge %d: source and target are required", i)
		}

		if edge.Channel == "" {
			c.Spec.Edges[i].Channel = "default"
		}

		if !nodeIDs[edge.SourceID] {
			return nil, nil, fmt.Errorf("edge %d: source node %s not found", i, edge.SourceID)
		}

		if !nodeIDs[edge.TargetID] {
			return nil, nil, fmt.Errorf("edge %d: target node %s not found", i, edge.TargetID)
		}

		if nodeTypeByID[edge.SourceID] == NodeTypeWidget {
			return nil, nil, fmt.Errorf("edge %d: widget nodes cannot be used as source nodes", i)
		}

		if nodeTypeByID[edge.TargetID] == NodeTypeWidget {
			return nil, nil, fmt.Errorf("edge %d: widget nodes cannot be used as target nodes", i)
		}

		if err := changesets.ValidateSourceNodeOutputChannel(
			registry,
			nodesByID[edge.SourceID],
			c.Spec.Edges[i].Channel,
		); err != nil {
			return nil, nil, fmt.Errorf("edge %d: %v", i, err)
		}
	}

	//
	// Check for cycles in the canvas
	//
	if err := changesets.CheckForCycles(c.Nodes(), c.Edges()); err != nil {
		return nil, nil, fmt.Errorf("invalid canvas graph: %v", err)
	}

	//
	// Return nodes enriched with errors and warnings found during validation
	//

	nodes := make([]models.Node, 0, len(c.Spec.Nodes))
	for _, node := range c.Spec.Nodes {
		n := node.Model()

		if errorMsg, hasError := nodeValidationErrors[node.ID]; hasError {
			n.ErrorMessage = &errorMsg
		} else {
			n.ErrorMessage = nil
		}

		if warningMsg, hasWarning := nodeWarnings[node.ID]; hasWarning {
			n.WarningMessage = &warningMsg
		} else {
			n.WarningMessage = nil
		}

		nodes = append(nodes, n)
	}

	return nodes, c.Edges(), nil
}

func (c *Canvas) validateNodeRef(registry *registry.Registry, organizationID string, node Node) error {
	if node.Component == "" {
		return fmt.Errorf("component name is required")
	}

	parts := strings.SplitN(node.Component, ".", 2)
	if len(parts) > 2 {
		return fmt.Errorf("invalid component name: %s", node.Component)
	}

	configurable, err := registry.FindConfigurableComponent(node.Component)
	if err != nil {
		return err
	}

	if len(parts) > 1 {
		err := c.validateIntegration(organizationID, node.Integration, node.Component)
		if err != nil {
			return err
		}
	}

	return configuration.ValidateConfiguration(configurable.Configuration(), node.Configuration)
}

func (c *Canvas) validateIntegration(organizationID string, ref *IntegrationRef, component string) error {
	if ref == nil || ref.ID == "" {
		return fmt.Errorf("integration is required")
	}

	integrationID, err := uuid.Parse(ref.ID)
	if err != nil {
		return fmt.Errorf("invalid integration ID: %v", err)
	}

	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return fmt.Errorf("invalid organization ID: %v", err)
	}

	integration, err := models.FindIntegration(orgID, integrationID)
	if err != nil {
		return fmt.Errorf("integration not found or does not belong to this organization")
	}

	if !integration.HasCapabilityEnabled(component) {
		return fmt.Errorf("%s is not enabled for integration %s", component, integration.InstallationName)
	}

	return nil
}

// FindShadowedNameWarnings detects nodes with duplicate names within connected components.
// Only nodes that are connected (directly or transitively) and share the same name will be flagged.
// Returns a map of node ID -> warning message.
func (c *Canvas) FindShadowedNameWarnings(registry *registry.Registry) map[string]string {
	warnings := make(map[string]string)

	if len(c.Spec.Nodes) == 0 {
		return warnings
	}

	// Build maps for node names and IDs
	nodeIDs := make(map[string]bool)
	nodeNameByID := make(map[string]string)

	for _, node := range c.Spec.Nodes {
		nodeType, err := registry.ComponentType(node.Component)
		if err != nil {
			continue
		}

		if nodeType == models.NodeTypeWidget {
			continue // Skip widgets
		}

		nodeIDs[node.ID] = true
		nodeNameByID[node.ID] = node.Name
	}

	// Find connected components using union-find
	parent := make(map[string]string)
	for id := range nodeIDs {
		parent[id] = id
	}

	var find func(x string) string
	find = func(x string) string {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y string) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	// Union nodes connected by edges
	for _, edge := range c.Spec.Edges {
		if edge.SourceID != "" && edge.TargetID != "" {
			// Only union if both nodes are tracked (non-widgets)
			if nodeIDs[edge.SourceID] && nodeIDs[edge.TargetID] {
				union(edge.SourceID, edge.TargetID)
			}
		}
	}

	// Group nodes by connected component
	componentNodes := make(map[string][]string)
	for id := range nodeIDs {
		root := find(id)
		componentNodes[root] = append(componentNodes[root], id)
	}

	// Check for shadowed names within each connected component
	for _, nodeIDsInComponent := range componentNodes {
		nameToIDs := make(map[string][]string)
		for _, nodeID := range nodeIDsInComponent {
			name := nodeNameByID[nodeID]
			nameToIDs[name] = append(nameToIDs[name], nodeID)
		}

		for name, ids := range nameToIDs {
			if len(ids) > 1 {
				warningMsg := "Multiple components named \"" + name + "\""
				for _, nodeID := range ids {
					warnings[nodeID] = warningMsg
				}
			}
		}
	}

	return warnings
}
