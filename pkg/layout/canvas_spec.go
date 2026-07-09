package layout

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// ApplyToCanvasSpec rearranges node positions in place.
func ApplyToCanvasSpec(spec *openapi_client.CanvasesCanvasSpec, autoLayout *AutoLayout) error {
	if spec == nil || autoLayout == nil {
		return nil
	}

	nodes := openAPINodesToModels(spec.GetNodes())
	edges := openAPIEdgesToModels(spec.GetEdges())
	layoutedNodes, _, err := ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return err
	}

	spec.SetNodes(applyLayoutPositions(spec.GetNodes(), layoutedNodes))
	return nil
}

func openAPINodesToModels(nodes []openapi_client.SuperplaneComponentsNode) []models.Node {
	result := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		nodeType, nodeRef := openAPINodeTypeAndRef(node)
		result = append(result, models.Node{
			ID:            node.GetId(),
			Name:          node.GetName(),
			Type:          nodeType,
			Ref:           nodeRef,
			Position:      openAPIPositionToModels(node.Position),
			IsCollapsed:   node.GetIsCollapsed(),
			IntegrationID: openAPIIntegrationID(node.Integration),
		})
	}
	return result
}

func openAPIEdgesToModels(edges []openapi_client.ComponentsEdge) []models.Edge {
	result := make([]models.Edge, 0, len(edges))
	for _, edge := range edges {
		result = append(result, models.Edge{
			SourceID: edge.GetSourceId(),
			TargetID: edge.GetTargetId(),
			Channel:  edge.GetChannel(),
		})
	}
	return result
}

func openAPINodeTypeAndRef(node openapi_client.SuperplaneComponentsNode) (string, models.NodeRef) {
	component := node.GetComponent()
	switch node.GetType() {
	case openapi_client.COMPONENTSNODETYPE_TYPE_TRIGGER:
		return models.NodeTypeTrigger, models.NodeRef{Trigger: &models.TriggerRef{Name: component}}
	case openapi_client.COMPONENTSNODETYPE_TYPE_WIDGET:
		return models.NodeTypeWidget, models.NodeRef{Widget: &models.WidgetRef{Name: component}}
	default:
		return models.NodeTypeComponent, models.NodeRef{Component: &models.ComponentRef{Name: component}}
	}
}

func openAPIPositionToModels(position *openapi_client.ComponentsPosition) models.Position {
	if position == nil {
		return models.Position{}
	}
	return models.Position{
		X: int(position.GetX()),
		Y: int(position.GetY()),
	}
}

func openAPIIntegrationID(integration *openapi_client.ComponentsIntegrationRef) *string {
	if integration == nil || !integration.HasId() {
		return nil
	}
	id := integration.GetId()
	return &id
}

func applyLayoutPositions(
	nodes []openapi_client.SuperplaneComponentsNode,
	layouted []models.Node,
) []openapi_client.SuperplaneComponentsNode {
	positionsByID := make(map[string]models.Position, len(layouted))
	for _, node := range layouted {
		if node.ID == "" {
			continue
		}
		positionsByID[node.ID] = node.Position
	}

	updated := make([]openapi_client.SuperplaneComponentsNode, len(nodes))
	copy(updated, nodes)

	for i := range updated {
		nodeID := updated[i].GetId()
		position, ok := positionsByID[nodeID]
		if !ok {
			continue
		}

		componentsPosition := openapi_client.NewComponentsPosition()
		componentsPosition.SetX(int32(position.X))
		componentsPosition.SetY(int32(position.Y))
		updated[i].SetPosition(*componentsPosition)
	}

	return updated
}

// ResolveUpdateAutoLayout picks the auto-layout settings for canvas update.
// Flags take precedence; otherwise a file-level autoLayout field is used.
// When neither is set, horizontal full-canvas layout is applied.
func ResolveUpdateAutoLayout(
	hasFlags bool,
	fileAutoLayout *AutoLayout,
	value string,
	scopeValue string,
	nodeIDs []string,
) (*AutoLayout, error) {
	if hasFlags {
		if fileAutoLayout != nil {
			return nil, fmt.Errorf("cannot use auto-layout flags with --file when file already defines autoLayout")
		}
		return ParseAutoLayout(value, scopeValue, nodeIDs)
	}
	if fileAutoLayout != nil {
		return fileAutoLayout, nil
	}
	defaultLayout := DefaultAutoLayout()
	return &defaultLayout, nil
}

func HasFlags(ctx core.CommandContext) bool {
	if ctx.Cmd == nil {
		return false
	}

	flags := ctx.Cmd.Flags()
	if flags == nil {
		return false
	}

	return flags.Changed("auto-layout") || flags.Changed("auto-layout-scope") || flags.Changed("auto-layout-node")
}
