package canvases

import (
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	autoLayoutHorizontalSpacing = 560
	autoLayoutVerticalSpacing   = 260
)

func applyCanvasAutoLayout(
	nodes []models.Node,
	edges []models.Edge,
	autoLayout *pb.CanvasAutoLayout,
) ([]models.Node, []models.Edge, error) {
	if autoLayout == nil {
		return nodes, edges, nil
	}

	switch autoLayout.Algorithm {
	case pb.CanvasAutoLayout_ALGORITHM_UNSPECIFIED:
		return nil, nil, status.Error(codes.InvalidArgument, "auto_layout.algorithm is required")
	case pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL:
		layoutedNodes, err := applyHorizontalAutoLayout(nodes, edges, autoLayout)
		if err != nil {
			return nil, nil, err
		}
		return layoutedNodes, edges, nil
	default:
		return nil, nil, status.Errorf(codes.InvalidArgument, "unsupported auto layout algorithm: %s", autoLayout.Algorithm.String())
	}
}

func applyHorizontalAutoLayout(nodes []models.Node, edges []models.Edge, autoLayout *pb.CanvasAutoLayout) ([]models.Node, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	nodeIndexByID := make(map[string]int, len(nodes))
	flowNodeIDs := make([]string, 0, len(nodes))
	flowNodeSet := make(map[string]struct{}, len(nodes))

	for i, node := range nodes {
		if node.ID == "" || node.Type == models.NodeTypeWidget {
			continue
		}

		nodeIndexByID[node.ID] = i
		flowNodeIDs = append(flowNodeIDs, node.ID)
		flowNodeSet[node.ID] = struct{}{}
	}

	if len(flowNodeIDs) == 0 {
		return nodes, nil
	}

	seedNodeIDs, err := resolveLayoutSeedNodeIDs(autoLayout, flowNodeSet)
	if err != nil {
		return nil, err
	}

	scope := resolveAutoLayoutScope(autoLayout, len(seedNodeIDs) > 0)
	selectedNodeIDs, err := resolveScopedNodeIDs(
		scope,
		seedNodeIDs,
		flowNodeIDs,
		flowNodeSet,
		nodeIndexByID,
		nodes,
		edges,
	)
	if err != nil {
		return nil, err
	}
	if len(selectedNodeIDs) == 0 {
		return nodes, nil
	}

	selectedNodeSet := make(map[string]struct{}, len(selectedNodeIDs))
	indegreeByID := make(map[string]int, len(selectedNodeIDs))
	outgoingByID := make(map[string][]string, len(selectedNodeIDs))
	incomingByID := make(map[string][]string, len(selectedNodeIDs))
	originalMinX := 0
	originalMinY := 0
	hasSelectedNodes := false

	for _, nodeID := range selectedNodeIDs {
		selectedNodeSet[nodeID] = struct{}{}
		indegreeByID[nodeID] = 0
		outgoingByID[nodeID] = []string{}
		incomingByID[nodeID] = []string{}

		node := nodes[nodeIndexByID[nodeID]]
		if !hasSelectedNodes {
			originalMinX = node.Position.X
			originalMinY = node.Position.Y
			hasSelectedNodes = true
			continue
		}

		if node.Position.X < originalMinX {
			originalMinX = node.Position.X
		}
		if node.Position.Y < originalMinY {
			originalMinY = node.Position.Y
		}
	}

	for _, edge := range edges {
		if _, ok := selectedNodeSet[edge.SourceID]; !ok {
			continue
		}
		if _, ok := selectedNodeSet[edge.TargetID]; !ok {
			continue
		}

		outgoingByID[edge.SourceID] = append(outgoingByID[edge.SourceID], edge.TargetID)
		incomingByID[edge.TargetID] = append(incomingByID[edge.TargetID], edge.SourceID)
		indegreeByID[edge.TargetID]++
	}

	for nodeID := range outgoingByID {
		sort.SliceStable(outgoingByID[nodeID], func(i, j int) bool {
			return nodeOrderLess(outgoingByID[nodeID][i], outgoingByID[nodeID][j], nodes, nodeIndexByID)
		})
	}

	queue := make([]string, 0, len(selectedNodeIDs))
	for _, nodeID := range selectedNodeIDs {
		if indegreeByID[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}
	sort.SliceStable(queue, func(i, j int) bool {
		return nodeOrderLess(queue[i], queue[j], nodes, nodeIndexByID)
	})

	order := make([]string, 0, len(selectedNodeIDs))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, target := range outgoingByID[current] {
			indegreeByID[target]--
			if indegreeByID[target] == 0 {
				queue = append(queue, target)
			}
		}

		sort.SliceStable(queue, func(i, j int) bool {
			return nodeOrderLess(queue[i], queue[j], nodes, nodeIndexByID)
		})
	}

	if len(order) < len(selectedNodeIDs) {
		inOrder := make(map[string]bool, len(order))
		for _, nodeID := range order {
			inOrder[nodeID] = true
		}

		remaining := make([]string, 0, len(selectedNodeIDs)-len(order))
		for _, nodeID := range selectedNodeIDs {
			if !inOrder[nodeID] {
				remaining = append(remaining, nodeID)
			}
		}

		sort.SliceStable(remaining, func(i, j int) bool {
			return nodeOrderLess(remaining[i], remaining[j], nodes, nodeIndexByID)
		})
		order = append(order, remaining...)
	}

	layerByID := make(map[string]int, len(selectedNodeIDs))
	maxLayer := 0
	for _, nodeID := range order {
		layer := 0
		for _, parentID := range incomingByID[nodeID] {
			parentLayer := layerByID[parentID] + 1
			if parentLayer > layer {
				layer = parentLayer
			}
		}
		layerByID[nodeID] = layer
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	nodesByLayer := make([][]string, maxLayer+1)
	for _, nodeID := range order {
		layer := layerByID[nodeID]
		nodesByLayer[layer] = append(nodesByLayer[layer], nodeID)
	}

	for layer := range nodesByLayer {
		sort.SliceStable(nodesByLayer[layer], func(i, j int) bool {
			return nodeOrderLess(nodesByLayer[layer][i], nodesByLayer[layer][j], nodes, nodeIndexByID)
		})
	}

	layoutedPositionByNodeID := make(map[string]models.Position, len(selectedNodeIDs))
	layoutMinX := 0
	layoutMinY := 0
	hasLayoutedNode := false

	for layer, layerNodeIDs := range nodesByLayer {
		x := layer * autoLayoutHorizontalSpacing
		for row, nodeID := range layerNodeIDs {
			position := models.Position{
				X: x,
				Y: row * autoLayoutVerticalSpacing,
			}
			layoutedPositionByNodeID[nodeID] = position

			if !hasLayoutedNode {
				layoutMinX = position.X
				layoutMinY = position.Y
				hasLayoutedNode = true
				continue
			}

			if position.X < layoutMinX {
				layoutMinX = position.X
			}
			if position.Y < layoutMinY {
				layoutMinY = position.Y
			}
		}
	}

	offsetX := originalMinX - layoutMinX
	offsetY := originalMinY - layoutMinY

	updatedNodes := make([]models.Node, len(nodes))
	copy(updatedNodes, nodes)

	for nodeID, position := range layoutedPositionByNodeID {
		index := nodeIndexByID[nodeID]
		updatedNodes[index].Position = models.Position{
			X: position.X + offsetX,
			Y: position.Y + offsetY,
		}
	}

	return updatedNodes, nil
}

func nodeOrderLess(nodeIDA string, nodeIDB string, nodes []models.Node, nodeIndexByID map[string]int) bool {
	indexA := nodeIndexByID[nodeIDA]
	indexB := nodeIndexByID[nodeIDB]
	nodeA := nodes[indexA]
	nodeB := nodes[indexB]

	if nodeA.Position.Y != nodeB.Position.Y {
		return nodeA.Position.Y < nodeB.Position.Y
	}
	if nodeA.Position.X != nodeB.Position.X {
		return nodeA.Position.X < nodeB.Position.X
	}

	return strings.Compare(nodeA.ID, nodeB.ID) < 0
}

func resolveLayoutSeedNodeIDs(autoLayout *pb.CanvasAutoLayout, flowNodeSet map[string]struct{}) ([]string, error) {
	if autoLayout == nil || len(autoLayout.NodeIds) == 0 {
		return []string{}, nil
	}

	seen := make(map[string]struct{}, len(autoLayout.NodeIds))
	seedNodeIDs := make([]string, 0, len(autoLayout.NodeIds))
	for _, nodeID := range autoLayout.NodeIds {
		if _, exists := flowNodeSet[nodeID]; !exists {
			return nil, status.Errorf(codes.InvalidArgument, "auto_layout.node_ids contains unknown node: %s", nodeID)
		}
		if _, exists := seen[nodeID]; exists {
			continue
		}
		seen[nodeID] = struct{}{}
		seedNodeIDs = append(seedNodeIDs, nodeID)
	}

	return seedNodeIDs, nil
}

func resolveAutoLayoutScope(autoLayout *pb.CanvasAutoLayout, hasSeedNodeIDs bool) pb.CanvasAutoLayout_Scope {
	if autoLayout == nil {
		return pb.CanvasAutoLayout_SCOPE_FULL_CANVAS
	}

	if autoLayout.Scope == pb.CanvasAutoLayout_SCOPE_UNSPECIFIED {
		if hasSeedNodeIDs {
			return pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT
		}
		return pb.CanvasAutoLayout_SCOPE_FULL_CANVAS
	}

	return autoLayout.Scope
}

func resolveScopedNodeIDs(
	scope pb.CanvasAutoLayout_Scope,
	seedNodeIDs []string,
	flowNodeIDs []string,
	flowNodeSet map[string]struct{},
	nodeIndexByID map[string]int,
	nodes []models.Node,
	edges []models.Edge,
) ([]string, error) {
	switch scope {
	case pb.CanvasAutoLayout_SCOPE_FULL_CANVAS:
		return cloneNodeIDs(flowNodeIDs), nil
	case pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT:
		return resolveConnectedComponentNodeIDs(seedNodeIDs, flowNodeIDs, flowNodeSet, nodeIndexByID, nodes, edges), nil
	case pb.CanvasAutoLayout_SCOPE_EXACT_SET:
		return resolveExactSetNodeIDs(seedNodeIDs)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported auto layout scope: %s", scope.String())
	}
}

func cloneNodeIDs(nodeIDs []string) []string {
	return append([]string(nil), nodeIDs...)
}

func resolveExactSetNodeIDs(seedNodeIDs []string) ([]string, error) {
	if len(seedNodeIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "auto_layout.node_ids is required when scope is EXACT_SET")
	}

	return cloneNodeIDs(seedNodeIDs), nil
}

func resolveConnectedComponentNodeIDs(
	seedNodeIDs []string,
	flowNodeIDs []string,
	flowNodeSet map[string]struct{},
	nodeIndexByID map[string]int,
	nodes []models.Node,
	edges []models.Edge,
) []string {
	if len(seedNodeIDs) == 0 {
		return cloneNodeIDs(flowNodeIDs)
	}

	adjacencyByNodeID := buildFlowAdjacency(flowNodeIDs, flowNodeSet, edges)
	sortAdjacencyByNodeOrder(adjacencyByNodeID, nodes, nodeIndexByID)

	selectedNodeSet := traverseConnectedNodeSet(seedNodeIDs, adjacencyByNodeID, len(flowNodeIDs))
	return collectSelectedNodeIDs(flowNodeIDs, selectedNodeSet)
}

func buildFlowAdjacency(
	flowNodeIDs []string,
	flowNodeSet map[string]struct{},
	edges []models.Edge,
) map[string][]string {
	adjacencyByNodeID := make(map[string][]string, len(flowNodeIDs))
	for _, nodeID := range flowNodeIDs {
		adjacencyByNodeID[nodeID] = []string{}
	}

	for _, edge := range edges {
		if _, ok := flowNodeSet[edge.SourceID]; !ok {
			continue
		}
		if _, ok := flowNodeSet[edge.TargetID]; !ok {
			continue
		}
		adjacencyByNodeID[edge.SourceID] = append(adjacencyByNodeID[edge.SourceID], edge.TargetID)
		adjacencyByNodeID[edge.TargetID] = append(adjacencyByNodeID[edge.TargetID], edge.SourceID)
	}

	return adjacencyByNodeID
}

func sortAdjacencyByNodeOrder(
	adjacencyByNodeID map[string][]string,
	nodes []models.Node,
	nodeIndexByID map[string]int,
) {
	for nodeID := range adjacencyByNodeID {
		sort.SliceStable(adjacencyByNodeID[nodeID], func(i, j int) bool {
			return nodeOrderLess(adjacencyByNodeID[nodeID][i], adjacencyByNodeID[nodeID][j], nodes, nodeIndexByID)
		})
	}
}

func traverseConnectedNodeSet(
	seedNodeIDs []string,
	adjacencyByNodeID map[string][]string,
	capacity int,
) map[string]struct{} {
	selectedNodeSet := make(map[string]struct{}, capacity)
	queue := make([]string, 0, len(seedNodeIDs))
	queue = append(queue, seedNodeIDs...)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, exists := selectedNodeSet[current]; exists {
			continue
		}
		selectedNodeSet[current] = struct{}{}

		for _, neighbor := range adjacencyByNodeID[current] {
			if _, exists := selectedNodeSet[neighbor]; exists {
				continue
			}
			queue = append(queue, neighbor)
		}
	}

	return selectedNodeSet
}

func collectSelectedNodeIDs(flowNodeIDs []string, selectedNodeSet map[string]struct{}) []string {
	selectedNodeIDs := make([]string, 0, len(selectedNodeSet))
	for _, nodeID := range flowNodeIDs {
		if _, exists := selectedNodeSet[nodeID]; exists {
			selectedNodeIDs = append(selectedNodeIDs, nodeID)
		}
	}

	return selectedNodeIDs
}
