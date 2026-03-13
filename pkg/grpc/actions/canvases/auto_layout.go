package canvases

import (
	"math"
	"sort"
	"strings"

	"github.com/nulab/autog"
	"github.com/nulab/autog/graph"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	autoLayoutNodeWidth          = 420.0
	autoLayoutNodeHeight         = 180.0
	autoLayoutLayerGap           = 180.0
	autoLayoutNodeGap            = 100.0
	autoLayoutUnknownChannelRank = 1 << 20
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

type incomingEdgeOrdering struct {
	parentID    string
	channel     string
	channelRank int
}

type nodeLayerPriority struct {
	parentOrder int
	channelRank int
	channel     string
}

// layoutContext holds shared state used across the layout pipeline.
type layoutContext struct {
	nodes                          []models.Node
	edges                          []models.Edge
	nodeIndexByID                  map[string]int
	selectedNodeSet                map[string]struct{}
	channelRankByNodeID            map[string]map[string]int
	incomingEdgeOrderingByTargetID map[string][]incomingEdgeOrdering
}

func applyHorizontalAutoLayout(
	nodes []models.Node,
	edges []models.Edge,
	autoLayout *pb.CanvasAutoLayout,
	registry *registry.Registry,
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

	lctx := &layoutContext{
		nodes:                          nodes,
		edges:                          edges,
		nodeIndexByID:                  nodeIndexByID,
		selectedNodeSet:                make(map[string]struct{}, len(selectedNodeIDs)),
		channelRankByNodeID:            buildOutputChannelRankByNodeID(nodes, nodeIndexByID, registry),
		incomingEdgeOrderingByTargetID: make(map[string][]incomingEdgeOrdering, len(selectedNodeIDs)),
	}

	originalMin := resolveOriginalMinPosition(lctx, selectedNodeIDs)
	autogEdges, connectedNodes := buildAutogEdges(lctx)
	isolatedNodeIDs := resolveIsolatedNodeIDs(selectedNodeIDs, connectedNodes)

	positions := computeAutogPositions(autogEdges)
	nodeOrderIndexByID := buildNodeOrderIndex(positions)
	applyChannelRankOrdering(lctx, positions, nodeOrderIndexByID)
	placeIsolatedNodes(lctx, positions, isolatedNodeIDs)
	applyPositionOffset(positions, originalMin)

	return positions.applyTo(nodes, nodeIndexByID), nil
}

// layoutPositions wraps the position map with helper methods.
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
		index := nodeIndexByID[nodeID]
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

func (lp *layoutPositions) maxX() int {
	maxX := 0
	for _, pos := range lp.byNodeID {
		if pos.X > maxX {
			maxX = pos.X
		}
	}
	return maxX
}

func resolveOriginalMinPosition(lctx *layoutContext, selectedNodeIDs []string) models.Position {
	originalMinX := 0
	originalMinY := 0
	first := true

	for _, nodeID := range selectedNodeIDs {
		lctx.selectedNodeSet[nodeID] = struct{}{}
		lctx.incomingEdgeOrderingByTargetID[nodeID] = []incomingEdgeOrdering{}

		node := lctx.nodes[lctx.nodeIndexByID[nodeID]]
		if first {
			originalMinX = node.Position.X
			originalMinY = node.Position.Y
			first = false
			continue
		}

		if node.Position.X < originalMinX {
			originalMinX = node.Position.X
		}
		if node.Position.Y < originalMinY {
			originalMinY = node.Position.Y
		}
	}

	return models.Position{X: originalMinX, Y: originalMinY}
}

func buildAutogEdges(lctx *layoutContext) ([][]string, map[string]struct{}) {
	var autogEdges [][]string
	connectedNodes := make(map[string]struct{})

	for _, edge := range lctx.edges {
		if _, ok := lctx.selectedNodeSet[edge.SourceID]; !ok {
			continue
		}
		if _, ok := lctx.selectedNodeSet[edge.TargetID]; !ok {
			continue
		}

		autogEdges = append(autogEdges, []string{edge.SourceID, edge.TargetID})
		connectedNodes[edge.SourceID] = struct{}{}
		connectedNodes[edge.TargetID] = struct{}{}

		lctx.incomingEdgeOrderingByTargetID[edge.TargetID] = append(
			lctx.incomingEdgeOrderingByTargetID[edge.TargetID],
			incomingEdgeOrdering{
				parentID:    edge.SourceID,
				channel:     edge.Channel,
				channelRank: resolveEdgeChannelRank(edge.SourceID, edge.Channel, lctx.channelRankByNodeID),
			},
		)
	}

	return autogEdges, connectedNodes
}

func resolveIsolatedNodeIDs(selectedNodeIDs []string, connectedNodes map[string]struct{}) []string {
	var isolated []string
	for _, id := range selectedNodeIDs {
		if _, ok := connectedNodes[id]; !ok {
			isolated = append(isolated, id)
		}
	}
	return isolated
}

// computeAutogPositions runs the Sugiyama layout via autog.
// autog is top-to-bottom, so we swap W↔H on input and X↔Y on output
// to produce a left-to-right horizontal layout.
func computeAutogPositions(autogEdges [][]string) *layoutPositions {
	positions := newLayoutPositions(0)

	if len(autogEdges) == 0 {
		return positions
	}

	result := autog.Layout(
		graph.EdgeSlice(autogEdges),
		autog.WithNodeFixedSize(autoLayoutNodeHeight, autoLayoutNodeWidth),
		autog.WithLayerSpacing(autoLayoutLayerGap),
		autog.WithNodeSpacing(autoLayoutNodeGap),
		autog.WithPositioning(autog.PositioningVAlign),
		autog.WithEdgeRouting(autog.EdgeRoutingNoop),
	)

	positions.byNodeID = make(map[string]models.Position, len(result.Nodes))
	for _, n := range result.Nodes {
		positions.byNodeID[n.ID] = models.Position{
			X: int(math.Round(n.Y)),
			Y: int(math.Round(n.X)),
		}
	}

	return positions
}

// buildNodeOrderIndex derives a topological order from autog's positions.
func buildNodeOrderIndex(positions *layoutPositions) map[string]int {
	type positionedNode struct {
		id string
		x  int
		y  int
	}

	ordered := make([]positionedNode, 0, len(positions.byNodeID))
	for id, pos := range positions.byNodeID {
		ordered = append(ordered, positionedNode{id: id, x: pos.X, y: pos.Y})
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].x != ordered[j].x {
			return ordered[i].x < ordered[j].x
		}
		return ordered[i].y < ordered[j].y
	})

	indexByID := make(map[string]int, len(ordered))
	for idx, pn := range ordered {
		indexByID[pn.id] = idx
	}
	return indexByID
}

// applyChannelRankOrdering re-sorts nodes within each layer by channel priority,
// then reassigns Y positions centered around 0.
func applyChannelRankOrdering(
	lctx *layoutContext,
	positions *layoutPositions,
	nodeOrderIndexByID map[string]int,
) {
	type layerGroup struct {
		x       int
		nodeIDs []string
	}

	layersByX := make(map[int]*layerGroup)
	for id, pos := range positions.byNodeID {
		lg, ok := layersByX[pos.X]
		if !ok {
			lg = &layerGroup{x: pos.X}
			layersByX[pos.X] = lg
		}
		lg.nodeIDs = append(lg.nodeIDs, id)
	}

	spacing := int(autoLayoutNodeHeight + autoLayoutNodeGap)
	for _, lg := range layersByX {
		sort.SliceStable(lg.nodeIDs, func(i, j int) bool {
			return layerNodeLess(
				lg.nodeIDs[i], lg.nodeIDs[j],
				lctx.incomingEdgeOrderingByTargetID,
				nodeOrderIndexByID,
				lctx.nodes, lctx.nodeIndexByID,
			)
		})

		totalHeight := (len(lg.nodeIDs) - 1) * spacing
		startY := -totalHeight / 2
		for i, id := range lg.nodeIDs {
			positions.byNodeID[id] = models.Position{
				X: lg.x,
				Y: startY + i*spacing,
			}
		}
	}
}

func layerNodeLess(
	nodeIDA, nodeIDB string,
	incomingEdgeOrderingByTargetID map[string][]incomingEdgeOrdering,
	nodeOrderIndexByID map[string]int,
	nodes []models.Node,
	nodeIndexByID map[string]int,
) bool {
	priorityA := resolveNodeLayerPriority(nodeIDA, incomingEdgeOrderingByTargetID, nodeOrderIndexByID)
	priorityB := resolveNodeLayerPriority(nodeIDB, incomingEdgeOrderingByTargetID, nodeOrderIndexByID)

	if priorityA.parentOrder != priorityB.parentOrder {
		return priorityA.parentOrder < priorityB.parentOrder
	}
	if priorityA.channelRank != priorityB.channelRank {
		return priorityA.channelRank < priorityB.channelRank
	}
	if priorityA.channel != priorityB.channel {
		return strings.Compare(priorityA.channel, priorityB.channel) < 0
	}

	return nodeOrderLess(nodeIDA, nodeIDB, nodes, nodeIndexByID)
}

func placeIsolatedNodes(
	lctx *layoutContext,
	positions *layoutPositions,
	isolatedNodeIDs []string,
) {
	if len(isolatedNodeIDs) == 0 {
		return
	}

	sort.SliceStable(isolatedNodeIDs, func(i, j int) bool {
		return nodeOrderLess(isolatedNodeIDs[i], isolatedNodeIDs[j], lctx.nodes, lctx.nodeIndexByID)
	})

	isolatedX := 0
	if len(positions.byNodeID) > 0 {
		isolatedX = positions.maxX() + int(autoLayoutNodeWidth+autoLayoutLayerGap)
	}

	spacing := int(autoLayoutNodeHeight + autoLayoutNodeGap)
	totalHeight := (len(isolatedNodeIDs) - 1) * spacing
	startY := -totalHeight / 2
	for i, id := range isolatedNodeIDs {
		positions.byNodeID[id] = models.Position{
			X: isolatedX,
			Y: startY + i*spacing,
		}
	}
}

func applyPositionOffset(
	positions *layoutPositions,
	originalMin models.Position,
) {
	if len(positions.byNodeID) == 0 {
		return
	}

	layoutMin := positions.minPosition()
	offsetX := originalMin.X - layoutMin.X
	offsetY := originalMin.Y - layoutMin.Y

	for id, pos := range positions.byNodeID {
		positions.byNodeID[id] = models.Position{
			X: pos.X + offsetX,
			Y: pos.Y + offsetY,
		}
	}
}

func nodeOrderLess(nodeIDA string, nodeIDB string, nodes []models.Node, nodeIndexByID map[string]int) bool {
	nodeA := nodes[nodeIndexByID[nodeIDA]]
	nodeB := nodes[nodeIndexByID[nodeIDB]]

	if nodeA.Position.Y != nodeB.Position.Y {
		return nodeA.Position.Y < nodeB.Position.Y
	}
	if nodeA.Position.X != nodeB.Position.X {
		return nodeA.Position.X < nodeB.Position.X
	}

	return strings.Compare(nodeA.ID, nodeB.ID) < 0
}

func buildOutputChannelRankByNodeID(
	nodes []models.Node,
	nodeIndexByID map[string]int,
	registry *registry.Registry,
) map[string]map[string]int {
	channelRankByNodeID := make(map[string]map[string]int, len(nodeIndexByID))
	if registry == nil {
		return channelRankByNodeID
	}

	for nodeID, index := range nodeIndexByID {
		node := nodes[index]
		if node.Ref.Component == nil || strings.TrimSpace(node.Ref.Component.Name) == "" {
			continue
		}

		component, err := registry.GetComponent(node.Ref.Component.Name)
		if err != nil {
			continue
		}

		outputChannels := component.OutputChannels(node.Configuration)
		if len(outputChannels) == 0 {
			outputChannels = []core.OutputChannel{core.DefaultOutputChannel}
		}

		channelRanks := make(map[string]int, len(outputChannels))
		for i, outputChannel := range outputChannels {
			channelName := strings.TrimSpace(outputChannel.Name)
			if channelName == "" {
				continue
			}
			channelRanks[channelName] = i
		}

		channelRankByNodeID[nodeID] = channelRanks
	}

	return channelRankByNodeID
}

func resolveEdgeChannelRank(
	sourceNodeID string,
	channel string,
	channelRankByNodeID map[string]map[string]int,
) int {
	channelRanks, ok := channelRankByNodeID[sourceNodeID]
	if !ok {
		return autoLayoutUnknownChannelRank
	}

	channelRank, ok := channelRanks[channel]
	if !ok {
		return autoLayoutUnknownChannelRank
	}

	return channelRank
}

func resolveNodeLayerPriority(
	nodeID string,
	incomingEdgeOrderingByTargetID map[string][]incomingEdgeOrdering,
	nodeOrderIndexByID map[string]int,
) nodeLayerPriority {
	incomingEdges := incomingEdgeOrderingByTargetID[nodeID]
	priority := nodeLayerPriority{
		parentOrder: len(nodeOrderIndexByID) + 1,
		channelRank: autoLayoutUnknownChannelRank,
		channel:     "",
	}

	for _, incomingEdge := range incomingEdges {
		parentOrder, exists := nodeOrderIndexByID[incomingEdge.parentID]
		if !exists {
			continue
		}

		if parentOrder < priority.parentOrder {
			priority.parentOrder = parentOrder
			priority.channelRank = incomingEdge.channelRank
			priority.channel = incomingEdge.channel
			continue
		}

		if parentOrder > priority.parentOrder {
			continue
		}

		if incomingEdge.channelRank < priority.channelRank {
			priority.channelRank = incomingEdge.channelRank
			priority.channel = incomingEdge.channel
			continue
		}

		if incomingEdge.channelRank > priority.channelRank {
			continue
		}

		if strings.Compare(incomingEdge.channel, priority.channel) < 0 {
			priority.channel = incomingEdge.channel
		}
	}

	return priority
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
