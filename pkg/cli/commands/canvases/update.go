package canvases

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file            *string
	draft           *bool
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}

	autoLayoutValue := ""
	if c.autoLayout != nil {
		autoLayoutValue = strings.TrimSpace(*c.autoLayout)
	}
	autoLayoutScopeValue := ""
	if c.autoLayoutScope != nil {
		autoLayoutScopeValue = strings.TrimSpace(*c.autoLayoutScope)
	}
	autoLayoutNodeIDs := []string{}
	if c.autoLayoutNodes != nil {
		autoLayoutNodeIDs = append(autoLayoutNodeIDs, *c.autoLayoutNodes...)
	}
	draftMode := c.draft != nil && *c.draft

	var (
		canvasID string
		canvas   openapi_client.CanvasesCanvas
		current  *openapi_client.CanvasesCanvas
		err      error
	)

	if filePath != "" {
		canvasID, canvas, err = loadCanvasFromFile(filePath)
		if err != nil {
			return err
		}
	} else {
		canvasID, canvas, err = loadCanvasFromExisting(ctx)
		if err != nil {
			return err
		}
		current = &canvas
	}

	versioningContext, err := resolveCanvasVersioningContext(ctx, canvasID)
	if err != nil {
		return err
	}

	targetVersionID := ""
	if !versioningContext.versioningEnabled {
		if draftMode {
			return fmt.Errorf("--draft cannot be used when effective canvas versioning is disabled")
		}
	} else {
		if !draftMode {
			return fmt.Errorf("effective canvas versioning is enabled for this canvas; use --draft")
		}

		targetVersionID, err = ensureCurrentUserDraftVersionID(ctx, canvasID)
		if err != nil {
			return err
		}
	}

	body := openapi_client.CanvasesUpdateCanvasVersionBody{}
	body.SetCanvas(canvas)
	if targetVersionID != "" {
		body.SetVersionId(targetVersionID)
	}

	if autoLayoutFlagsWereSet(ctx) {
		if autoLayoutValue == "" && (autoLayoutScopeValue != "" || len(autoLayoutNodeIDs) > 0) {
			return fmt.Errorf("--auto-layout is required when using --auto-layout-scope or --auto-layout-node")
		}

		if autoLayoutValue != "" {
			autoLayout, parseErr := parseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
			if parseErr != nil {
				return parseErr
			}
			body.SetAutoLayout(*autoLayout)
		}
	} else {
		if current == nil {
			if targetVersionID != "" {
				version, describeErr := describeCanvasVersionByID(ctx, canvasID, targetVersionID)
				if describeErr != nil {
					return describeErr
				}

				versionCanvas := canvasFromVersion(version)
				current = &versionCanvas
			} else {
				existingCanvas, describeErr := describeCanvasByID(ctx, canvasID)
				if describeErr != nil {
					return describeErr
				}
				current = &existingCanvas
			}
		}

		body.SetAutoLayout(buildDefaultAutoLayout(*current, canvas))
	}

	_, _, err = ctx.API.CanvasVersionAPI.
		CanvasesUpdateCanvasVersion2(ctx.Context, canvasID).
		Body(body).
		Execute()
	return err
}

func loadCanvasFromFile(filePath string) (string, openapi_client.CanvasesCanvas, error) {
	// #nosec
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("failed to read resource file: %w", err)
	}

	_, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	if kind != models.CanvasKind {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("unsupported resource kind %q for update", kind)
	}

	resource, err := models.ParseCanvas(data)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}
	if resource.Metadata == nil || resource.Metadata.Id == nil || resource.Metadata.GetId() == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata.id is required for update")
	}

	return resource.Metadata.GetId(), models.CanvasFromCanvas(*resource), nil
}

func loadCanvasFromExisting(ctx core.CommandContext) (string, openapi_client.CanvasesCanvas, error) {
	if len(ctx.Args) > 1 {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("update accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = ctx.Args[0]
	} else if ctx.Config != nil {
		target = strings.TrimSpace(ctx.Config.GetActiveCanvas())
	}

	if target == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("either --file or <name-or-id> (or an active canvas) is required")
	}

	canvasID, err := findCanvasID(ctx, ctx.API, target)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	canvas, err := describeCanvasByID(ctx, canvasID)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	return canvasID, canvas, nil
}

func parseAutoLayout(value string, scopeValue string, nodeIDs []string) (*openapi_client.CanvasesCanvasAutoLayout, error) {
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "horizontal":
		autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	default:
		return nil, fmt.Errorf("unsupported auto layout %q (supported: horizontal)", value)
	}

	normalizedNodeIDs := make([]string, 0, len(nodeIDs))
	seen := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		trimmed := strings.TrimSpace(nodeID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalizedNodeIDs = append(normalizedNodeIDs, trimmed)
	}
	if len(normalizedNodeIDs) > 0 {
		autoLayout.SetNodeIds(normalizedNodeIDs)
	}

	if strings.TrimSpace(scopeValue) == "" {
		return &autoLayout, nil
	}

	switch strings.ToLower(strings.TrimSpace(scopeValue)) {
	case "full-canvas", "full_canvas", "full":
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	case "connected-component", "connected_component", "connected":
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT)
	default:
		return nil, fmt.Errorf("unsupported auto layout scope %q (supported: full-canvas, connected-component)", scopeValue)
	}

	return &autoLayout, nil
}

func autoLayoutFlagsWereSet(ctx core.CommandContext) bool {
	if ctx.Cmd == nil {
		return false
	}

	flags := ctx.Cmd.Flags()
	if flags == nil {
		return false
	}

	return flags.Changed("auto-layout") || flags.Changed("auto-layout-scope") || flags.Changed("auto-layout-node")
}

func describeCanvasByID(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvas, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}
	if response.Canvas == nil {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas %q not found", canvasID)
	}

	return *response.Canvas, nil
}

func buildDefaultAutoLayout(current openapi_client.CanvasesCanvas, next openapi_client.CanvasesCanvas) openapi_client.CanvasesCanvasAutoLayout {
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}
	autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)

	changedFlowNodeIDs := resolveChangedFlowNodeIDs(current, next)
	if len(changedFlowNodeIDs) == 0 {
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
		return autoLayout
	}

	autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT)
	autoLayout.SetNodeIds(changedFlowNodeIDs)
	return autoLayout
}

func resolveChangedFlowNodeIDs(current openapi_client.CanvasesCanvas, next openapi_client.CanvasesCanvas) []string {
	currentSpec := current.GetSpec()
	nextSpec := next.GetSpec()

	currentNodesByID := mapNodesByID(currentSpec.GetNodes())
	nextNodesByID := mapNodesByID(nextSpec.GetNodes())

	changedNodeIDs := make(map[string]struct{})

	for _, nextNode := range nextSpec.GetNodes() {
		nodeID := strings.TrimSpace(nextNode.GetId())
		if nodeID == "" {
			continue
		}

		currentNode, exists := currentNodesByID[nodeID]
		if !exists {
			addChangedNodeIDIfFlow(changedNodeIDs, nodeID, nextNodesByID)
			continue
		}

		if canvasNodesDifferForAutoLayout(currentNode, nextNode) {
			addChangedNodeIDIfFlow(changedNodeIDs, nodeID, nextNodesByID)
		}
	}

	for nodeID := range currentNodesByID {
		if _, exists := nextNodesByID[nodeID]; exists {
			continue
		}

		for _, edge := range currentSpec.GetEdges() {
			sourceID := strings.TrimSpace(edge.GetSourceId())
			targetID := strings.TrimSpace(edge.GetTargetId())

			if sourceID == nodeID {
				addChangedNodeIDIfFlow(changedNodeIDs, targetID, nextNodesByID)
			}
			if targetID == nodeID {
				addChangedNodeIDIfFlow(changedNodeIDs, sourceID, nextNodesByID)
			}
		}
	}

	currentEdgesByKey := mapEdgesByKey(currentSpec.GetEdges())
	nextEdgesByKey := mapEdgesByKey(nextSpec.GetEdges())

	for key, edge := range nextEdgesByKey {
		if _, exists := currentEdgesByKey[key]; exists {
			continue
		}
		addChangedNodeIDIfFlow(changedNodeIDs, strings.TrimSpace(edge.GetSourceId()), nextNodesByID)
		addChangedNodeIDIfFlow(changedNodeIDs, strings.TrimSpace(edge.GetTargetId()), nextNodesByID)
	}

	for key, edge := range currentEdgesByKey {
		if _, exists := nextEdgesByKey[key]; exists {
			continue
		}
		addChangedNodeIDIfFlow(changedNodeIDs, strings.TrimSpace(edge.GetSourceId()), nextNodesByID)
		addChangedNodeIDIfFlow(changedNodeIDs, strings.TrimSpace(edge.GetTargetId()), nextNodesByID)
	}

	orderedNodeIDs := make([]string, 0, len(changedNodeIDs))
	for _, nextNode := range nextSpec.GetNodes() {
		nodeID := strings.TrimSpace(nextNode.GetId())
		if nodeID == "" {
			continue
		}
		if _, exists := changedNodeIDs[nodeID]; !exists {
			continue
		}
		orderedNodeIDs = append(orderedNodeIDs, nodeID)
	}

	return orderedNodeIDs
}

func addChangedNodeIDIfFlow(
	changedNodeIDs map[string]struct{},
	nodeID string,
	nextNodesByID map[string]openapi_client.ComponentsNode,
) {
	if nodeID == "" {
		return
	}

	node, exists := nextNodesByID[nodeID]
	if !exists {
		return
	}
	if node.GetType() == openapi_client.COMPONENTSNODETYPE_TYPE_WIDGET {
		return
	}

	changedNodeIDs[nodeID] = struct{}{}
}

func canvasNodesDifferForAutoLayout(current openapi_client.ComponentsNode, next openapi_client.ComponentsNode) bool {
	normalizedCurrent := current
	normalizedCurrent.ErrorMessage = nil
	normalizedCurrent.WarningMessage = nil

	normalizedNext := next
	normalizedNext.ErrorMessage = nil
	normalizedNext.WarningMessage = nil

	return !reflect.DeepEqual(normalizedCurrent, normalizedNext)
}

func mapNodesByID(nodes []openapi_client.ComponentsNode) map[string]openapi_client.ComponentsNode {
	nodesByID := make(map[string]openapi_client.ComponentsNode, len(nodes))
	for _, node := range nodes {
		nodeID := strings.TrimSpace(node.GetId())
		if nodeID == "" {
			continue
		}
		nodesByID[nodeID] = node
	}
	return nodesByID
}

func mapEdgesByKey(edges []openapi_client.ComponentsEdge) map[string]openapi_client.ComponentsEdge {
	edgesByKey := make(map[string]openapi_client.ComponentsEdge, len(edges))
	for _, edge := range edges {
		edgeKey := strings.Join([]string{
			strings.TrimSpace(edge.GetSourceId()),
			strings.TrimSpace(edge.GetTargetId()),
			strings.TrimSpace(edge.GetChannel()),
		}, "\x00")
		edgesByKey[edgeKey] = edge
	}
	return edgesByKey
}
