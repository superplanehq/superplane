package canvases

import (
	"math"
	"sort"
	"strings"

	"github.com/nulab/autog"
	"github.com/nulab/autog/graph"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	autoLayoutNodeWidth                        = 420.0
	autoLayoutNodeHeight                       = 180.0
	autoLayoutLayerGap                         = 220.0
	autoLayoutNodeGap                          = 180.0
	autoLayoutDisconnectedComponentVerticalGap = 280
)

func applyCanvasAutoLayout(
	nodes []models.Node,
	edges []models.Edge,
	autoLayout *pb.CanvasAutoLayout,
	registry *registry.Registry,
) ([]models.Node, []models.Edge, error) {
	if autoLayout == nil {
		return nodes, edges, nil
	}

	switch autoLayout.Algorithm {
	case pb.CanvasAutoLayout_ALGORITHM_UNSPECIFIED:
		return nil, nil, status.Error(codes.InvalidArgument, "auto_layout.algorithm is required")
	case pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL:
		layoutedNodes, err := applyHorizontalAutoLayout(nodes, edges, autoLayout, registry)
		if err != nil {
			return nil, nil, err
		}
		return layoutedNodes, edges, nil
	default:
		return nil, nil, status.Errorf(codes.InvalidArgument, "unsupported auto layout algorithm: %s", autoLayout.Algorithm.String())
	}
}

func applyHorizontalAutoLayout(
	nodes []models.Node,
	edges []models.Edge,
	autoLayout *pb.CanvasAutoLayout,
	_ *registry.Registry,
) ([]models.Node, error) {
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
	scopedNodeIDs, err := resolveScopedNodeIDs(scope, seedNodeIDs, flowNodeIDs, flowNodeSet, edges)
	if err != nil {
		return nil, err
	}
	if len(scopedNodeIDs) == 0 {
		return nodes, nil
	}

	layoutNodes := resolveLayoutNodes(nodes, nodeIndexByID, scopedNodeIDs)
	if len(layoutNodes) == 0 {
		return nodes, nil
	}

	layoutNodeSet := make(map[string]struct{}, len(layoutNodes))
	for _, node := range layoutNodes {
		layoutNodeSet[node.ID] = struct{}{}
	}

	layoutEdges := resolveLayoutEdges(edges, layoutNodeSet)
	components := resolveDisconnectedLayoutComponents(layoutNodes, layoutEdges)
	if len(components) == 0 {
		return nodes, nil
	}

	sortedComponents := sortComponentsByCurrentPosition(components)
	layoutedPositions := resolvePackedLayoutedPositions(sortedComponents, layoutEdges)
	if len(layoutedPositions.byNodeID) == 0 {
		return nodes, nil
	}

	minCurrentPosition := resolveMinPositionFromNodes(layoutNodes)
	minLayoutPosition := resolveMinPositionFromLayout(layoutedPositions)
	applyPositionOffset(layoutedPositions, models.Position{
		X: minCurrentPosition.X - minLayoutPosition.X,
		Y: minCurrentPosition.Y - minLayoutPosition.Y,
	})

	return layoutedPositions.applyTo(nodes, nodeIndexByID), nil
}

type layoutPositions struct {
	byNodeID map[string]models.Position
}

func newLayoutPositions(capacity int) *layoutPositions {
	return &layoutPositions{byNodeID: make(map[string]models.Position, capacity)}
}

func (lp *layoutPositions) applyTo(nodes []models.Node, nodeIndexByID map[string]int) []models.Node {
	updatedNodes := make([]models.Node, len(nodes))
	copy(updatedNodes, nodes)

	for nodeID, position := range lp.byNodeID {
		index, exists := nodeIndexByID[nodeID]
		if !exists {
			continue
		}
		updatedNodes[index].Position = position
	}

	return updatedNodes
}

func (lp *layoutPositions) minPosition() models.Position {
	minX := 0
	minY := 0
	first := true
	for _, pos := range lp.byNodeID {
		if first {
			minX = pos.X
			minY = pos.Y
			first = false
			continue
		}

		if pos.X < minX {
			minX = pos.X
		}
		if pos.Y < minY {
			minY = pos.Y
		}
	}

	return models.Position{X: minX, Y: minY}
}

func resolveLayoutNodes(nodes []models.Node, nodeIndexByID map[string]int, scopedNodeIDs []string) []models.Node {
	layoutNodes := make([]models.Node, 0, len(scopedNodeIDs))
	for _, nodeID := range scopedNodeIDs {
		index, exists := nodeIndexByID[nodeID]
		if !exists {
			continue
		}
		layoutNodes = append(layoutNodes, nodes[index])
	}
	return layoutNodes
}

func resolveLayoutEdges(edges []models.Edge, nodeSet map[string]struct{}) []models.Edge {
	layoutEdges := make([]models.Edge, 0, len(edges))
	for _, edge := range edges {
		if _, ok := nodeSet[edge.SourceID]; !ok {
			continue
		}
		if _, ok := nodeSet[edge.TargetID]; !ok {
			continue
		}
		layoutEdges = append(layoutEdges, edge)
	}
	return layoutEdges
}

func resolveDisconnectedLayoutComponents(layoutNodes []models.Node, layoutEdges []models.Edge) [][]models.Node {
	if len(layoutNodes) == 0 {
		return [][]models.Node{}
	}

	nodesByID := make(map[string]models.Node, len(layoutNodes))
	adjacencyByNodeID := make(map[string][]string, len(layoutNodes))
	layoutNodeSet := make(map[string]struct{}, len(layoutNodes))

	for _, node := range layoutNodes {
		nodesByID[node.ID] = node
		adjacencyByNodeID[node.ID] = []string{}
		layoutNodeSet[node.ID] = struct{}{}
	}

	for _, edge := range layoutEdges {
		if _, ok := layoutNodeSet[edge.SourceID]; !ok {
			continue
		}
		if _, ok := layoutNodeSet[edge.TargetID]; !ok {
			continue
		}

		adjacencyByNodeID[edge.SourceID] = append(adjacencyByNodeID[edge.SourceID], edge.TargetID)
		adjacencyByNodeID[edge.TargetID] = append(adjacencyByNodeID[edge.TargetID], edge.SourceID)
	}

	visitedNodeIDs := make(map[string]struct{}, len(layoutNodes))
	components := make([][]models.Node, 0)

	for _, node := range layoutNodes {
		seedNodeID := node.ID
		if _, visited := visitedNodeIDs[seedNodeID]; visited {
			continue
		}

		queue := []string{seedNodeID}
		componentNodes := make([]models.Node, 0)

		for len(queue) > 0 {
			currentNodeID := queue[0]
			queue = queue[1:]

			if _, visited := visitedNodeIDs[currentNodeID]; visited {
				continue
			}

			visitedNodeIDs[currentNodeID] = struct{}{}
			currentNode, exists := nodesByID[currentNodeID]
			if exists {
				componentNodes = append(componentNodes, currentNode)
			}

			neighbors := adjacencyByNodeID[currentNodeID]
			for _, neighborNodeID := range neighbors {
				if _, visited := visitedNodeIDs[neighborNodeID]; visited {
					continue
				}
				queue = append(queue, neighborNodeID)
			}
		}

		if len(componentNodes) > 0 {
			components = append(components, componentNodes)
		}
	}

	return components
}

func sortComponentsByCurrentPosition(components [][]models.Node) [][]models.Node {
	sorted := append([][]models.Node(nil), components...)

	sort.SliceStable(sorted, func(i, j int) bool {
		minA := resolveMinPositionFromNodes(sorted[i])
		minB := resolveMinPositionFromNodes(sorted[j])
		if minA.Y != minB.Y {
			return minA.Y < minB.Y
		}
		return minA.X < minB.X
	})

	return sorted
}

func resolvePackedLayoutedPositions(sortedComponents [][]models.Node, layoutEdges []models.Edge) *layoutPositions {
	totalNodes := 0
	for _, component := range sortedComponents {
		totalNodes += len(component)
	}

	packedPositions := newLayoutPositions(totalNodes)
	if len(sortedComponents) == 0 {
		return packedPositions
	}

	if len(sortedComponents) == 1 {
		nodeSet := make(map[string]struct{}, len(sortedComponents[0]))
		for _, node := range sortedComponents[0] {
			nodeSet[node.ID] = struct{}{}
		}
		componentEdges := resolveLayoutEdges(layoutEdges, nodeSet)
		return computeAutogPositions(sortedComponents[0], componentEdges)
	}

	currentTopY := 0
	for _, componentNodes := range sortedComponents {
		nodeSet := make(map[string]struct{}, len(componentNodes))
		for _, node := range componentNodes {
			nodeSet[node.ID] = struct{}{}
		}

		componentEdges := resolveLayoutEdges(layoutEdges, nodeSet)
		componentPositions := computeAutogPositions(componentNodes, componentEdges)
		if len(componentPositions.byNodeID) == 0 {
			continue
		}

		bounds := resolveLayoutBounds(componentNodes, componentPositions)
		for nodeID, position := range componentPositions.byNodeID {
			packedPositions.byNodeID[nodeID] = models.Position{
				X: position.X - bounds.minX,
				Y: position.Y - bounds.minY + currentTopY,
			}
		}

		currentTopY += bounds.height + autoLayoutDisconnectedComponentVerticalGap
	}

	return packedPositions
}

type layoutBounds struct {
	minX   int
	minY   int
	maxX   int
	maxY   int
	width  int
	height int
}

func resolveLayoutBounds(componentNodes []models.Node, componentPositions *layoutPositions) layoutBounds {
	bounds := layoutBounds{}
	first := true

	for _, node := range componentNodes {
		position, exists := componentPositions.byNodeID[node.ID]
		if !exists {
			continue
		}

		nodeMaxX := position.X + int(autoLayoutNodeWidth)
		nodeMaxY := position.Y + int(autoLayoutNodeHeight)

		if first {
			bounds.minX = position.X
			bounds.minY = position.Y
			bounds.maxX = nodeMaxX
			bounds.maxY = nodeMaxY
			first = false
			continue
		}

		if position.X < bounds.minX {
			bounds.minX = position.X
		}
		if position.Y < bounds.minY {
			bounds.minY = position.Y
		}
		if nodeMaxX > bounds.maxX {
			bounds.maxX = nodeMaxX
		}
		if nodeMaxY > bounds.maxY {
			bounds.maxY = nodeMaxY
		}
	}

	if first {
		return layoutBounds{}
	}

	bounds.width = bounds.maxX - bounds.minX
	bounds.height = bounds.maxY - bounds.minY
	return bounds
}

// computeAutogPositions runs the Sugiyama layout via autog.
// autog is top-to-bottom, so we swap W↔H on input and X↔Y on output
// to produce a left-to-right horizontal layout.
func computeAutogPositions(componentNodes []models.Node, componentEdges []models.Edge) *layoutPositions {
	positions := newLayoutPositions(len(componentNodes))
	if len(componentNodes) == 0 {
		return positions
	}

	if len(componentEdges) == 0 {
		if len(componentNodes) == 1 {
			positions.byNodeID[componentNodes[0].ID] = models.Position{X: 0, Y: 0}
			return positions
		}

		nodes := append([]models.Node(nil), componentNodes...)
		sort.SliceStable(nodes, func(i, j int) bool {
			if nodes[i].Position.Y != nodes[j].Position.Y {
				return nodes[i].Position.Y < nodes[j].Position.Y
			}
			if nodes[i].Position.X != nodes[j].Position.X {
				return nodes[i].Position.X < nodes[j].Position.X
			}
			return strings.Compare(nodes[i].ID, nodes[j].ID) < 0
		})

		spacing := int(autoLayoutNodeHeight + autoLayoutNodeGap)
		for i, node := range nodes {
			positions.byNodeID[node.ID] = models.Position{X: 0, Y: i * spacing}
		}
		return positions
	}

	autogEdges := make([][]string, 0, len(componentEdges))
	seenAutogEdges := make(map[string]struct{}, len(componentEdges))
	for _, edge := range componentEdges {
		key := edge.SourceID + "->" + edge.TargetID
		if _, exists := seenAutogEdges[key]; exists {
			continue
		}
		seenAutogEdges[key] = struct{}{}
		autogEdges = append(autogEdges, []string{edge.SourceID, edge.TargetID})
	}

	result := autog.Layout(
		graph.EdgeSlice(autogEdges),
		autog.WithNodeFixedSize(autoLayoutNodeHeight, autoLayoutNodeWidth),
		autog.WithLayerSpacing(autoLayoutLayerGap),
		autog.WithNodeSpacing(autoLayoutNodeGap),
		autog.WithPositioning(autog.PositioningVAlign),
		autog.WithEdgeRouting(autog.EdgeRoutingNoop),
	)

	for _, n := range result.Nodes {
		positions.byNodeID[n.ID] = models.Position{
			X: int(math.Round(n.Y)),
			Y: int(math.Round(n.X)),
		}
	}

	if len(positions.byNodeID) == len(componentNodes) {
		return positions
	}

	missingNodes := make([]models.Node, 0)
	for _, node := range componentNodes {
		if _, exists := positions.byNodeID[node.ID]; exists {
			continue
		}
		missingNodes = append(missingNodes, node)
	}
	if len(missingNodes) == 0 {
		return positions
	}

	sort.SliceStable(missingNodes, func(i, j int) bool {
		if missingNodes[i].Position.Y != missingNodes[j].Position.Y {
			return missingNodes[i].Position.Y < missingNodes[j].Position.Y
		}
		if missingNodes[i].Position.X != missingNodes[j].Position.X {
			return missingNodes[i].Position.X < missingNodes[j].Position.X
		}
		return strings.Compare(missingNodes[i].ID, missingNodes[j].ID) < 0
	})

	spacing := int(autoLayoutNodeHeight + autoLayoutNodeGap)
	startY := 0
	if len(positions.byNodeID) > 0 {
		maxY := 0
		first := true
		for _, pos := range positions.byNodeID {
			if first || pos.Y > maxY {
				maxY = pos.Y
				first = false
			}
		}
		startY = maxY + spacing
	}

	for i, node := range missingNodes {
		positions.byNodeID[node.ID] = models.Position{X: 0, Y: startY + i*spacing}
	}

	return positions
}

func resolveMinPositionFromNodes(nodes []models.Node) models.Position {
	minX := 0
	minY := 0
	first := true

	for _, node := range nodes {
		if first {
			minX = node.Position.X
			minY = node.Position.Y
			first = false
			continue
		}

		if node.Position.X < minX {
			minX = node.Position.X
		}
		if node.Position.Y < minY {
			minY = node.Position.Y
		}
	}

	return models.Position{X: minX, Y: minY}
}

func resolveMinPositionFromLayout(layoutedPositions *layoutPositions) models.Position {
	if len(layoutedPositions.byNodeID) == 0 {
		return models.Position{}
	}
	return layoutedPositions.minPosition()
}

func applyPositionOffset(positions *layoutPositions, offset models.Position) {
	for nodeID, pos := range positions.byNodeID {
		positions.byNodeID[nodeID] = models.Position{
			X: pos.X + offset.X,
			Y: pos.Y + offset.Y,
		}
	}
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
	edges []models.Edge,
) ([]string, error) {
	switch scope {
	case pb.CanvasAutoLayout_SCOPE_FULL_CANVAS:
		return cloneNodeIDs(flowNodeIDs), nil
	case pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT:
		return resolveConnectedComponentNodeIDs(seedNodeIDs, flowNodeIDs, flowNodeSet, edges), nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported auto layout scope: %s", scope.String())
	}
}

func cloneNodeIDs(nodeIDs []string) []string {
	return append([]string(nil), nodeIDs...)
}

func resolveConnectedComponentNodeIDs(
	seedNodeIDs []string,
	flowNodeIDs []string,
	flowNodeSet map[string]struct{},
	edges []models.Edge,
) []string {
	if len(seedNodeIDs) == 0 {
		return cloneNodeIDs(flowNodeIDs)
	}

	adjacencyByNodeID := buildFlowAdjacency(flowNodeIDs, flowNodeSet, edges)
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
