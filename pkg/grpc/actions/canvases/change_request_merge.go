package canvases

import "github.com/superplanehq/superplane/pkg/models"

func mergeCanvasVersionIntoLive(
	baseNodes []models.Node,
	baseEdges []models.Edge,
	liveNodes []models.Node,
	liveEdges []models.Edge,
	versionNodes []models.Node,
	versionEdges []models.Edge,
	changedNodeIDs []string,
) ([]models.Node, []models.Edge) {
	changedSet := make(map[string]struct{}, len(changedNodeIDs))
	for _, nodeID := range changedNodeIDs {
		if nodeID == "" {
			continue
		}
		changedSet[nodeID] = struct{}{}
	}

	mergedNodes := mergeCanvasNodesForChangeRequest(baseNodes, liveNodes, versionNodes, changedSet)
	mergedNodeSet := make(map[string]struct{}, len(mergedNodes))
	for _, node := range mergedNodes {
		if node.ID == "" {
			continue
		}
		mergedNodeSet[node.ID] = struct{}{}
	}

	mergedEdges := mergeCanvasEdgesForChangeRequest(liveEdges, versionEdges, changedSet, mergedNodeSet)
	return mergedNodes, mergedEdges
}

func mergeCanvasNodesForChangeRequest(
	baseNodes []models.Node,
	liveNodes []models.Node,
	versionNodes []models.Node,
	changedSet map[string]struct{},
) []models.Node {
	if len(changedSet) == 0 {
		return append([]models.Node(nil), liveNodes...)
	}

	baseByID := mapNodesByID(baseNodes)
	versionByID := mapNodesByID(versionNodes)

	merged := append([]models.Node(nil), liveNodes...)
	mergedIndexByID := make(map[string]int, len(merged))
	for i, node := range merged {
		if node.ID == "" {
			continue
		}
		mergedIndexByID[node.ID] = i
	}

	for _, node := range versionNodes {
		if node.ID == "" {
			continue
		}
		if _, ok := changedSet[node.ID]; !ok {
			continue
		}

		if existingIdx, ok := mergedIndexByID[node.ID]; ok {
			merged[existingIdx] = node
			continue
		}

		merged = append(merged, node)
		mergedIndexByID[node.ID] = len(merged) - 1
	}

	deletions := make(map[string]struct{})
	for nodeID := range changedSet {
		_, hasVersion := versionByID[nodeID]
		_, hasBase := baseByID[nodeID]
		if !hasVersion && hasBase {
			deletions[nodeID] = struct{}{}
		}
	}

	if len(deletions) == 0 {
		return merged
	}

	result := make([]models.Node, 0, len(merged))
	for _, node := range merged {
		if _, shouldDelete := deletions[node.ID]; shouldDelete {
			continue
		}
		result = append(result, node)
	}

	return result
}

func mergeCanvasEdgesForChangeRequest(
	liveEdges []models.Edge,
	versionEdges []models.Edge,
	changedSet map[string]struct{},
	mergedNodeSet map[string]struct{},
) []models.Edge {
	if len(changedSet) == 0 {
		return append([]models.Edge(nil), liveEdges...)
	}

	result := make([]models.Edge, 0, len(liveEdges)+len(versionEdges))
	added := make(map[string]struct{}, len(liveEdges)+len(versionEdges))

	for _, edge := range liveEdges {
		if edgeTouchesChangedNode(edge, changedSet) {
			continue
		}
		if !edgeReferencesKnownNodes(edge, mergedNodeSet) {
			continue
		}

		edgeKey := edge.SourceID + "|" + edge.TargetID + "|" + edge.Channel
		if _, ok := added[edgeKey]; ok {
			continue
		}
		added[edgeKey] = struct{}{}
		result = append(result, edge)
	}

	for _, edge := range versionEdges {
		if !edgeTouchesChangedNode(edge, changedSet) {
			continue
		}
		if !edgeReferencesKnownNodes(edge, mergedNodeSet) {
			continue
		}

		edgeKey := edge.SourceID + "|" + edge.TargetID + "|" + edge.Channel
		if _, ok := added[edgeKey]; ok {
			continue
		}
		added[edgeKey] = struct{}{}
		result = append(result, edge)
	}

	return result
}

func edgeTouchesChangedNode(edge models.Edge, changedSet map[string]struct{}) bool {
	if _, ok := changedSet[edge.SourceID]; ok {
		return true
	}
	if _, ok := changedSet[edge.TargetID]; ok {
		return true
	}
	return false
}

func edgeReferencesKnownNodes(edge models.Edge, nodeSet map[string]struct{}) bool {
	if edge.SourceID == "" || edge.TargetID == "" {
		return false
	}

	if _, ok := nodeSet[edge.SourceID]; !ok {
		return false
	}
	if _, ok := nodeSet[edge.TargetID]; !ok {
		return false
	}
	return true
}
