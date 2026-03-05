package canvases

import (
	"reflect"
	"sort"

	"github.com/superplanehq/superplane/pkg/models"
)

type comparableCanvasNode struct {
	ID            string
	Name          string
	Type          string
	Ref           models.NodeRef
	Configuration map[string]any
	Position      models.Position
	IsCollapsed   bool
	IntegrationID *string
}

func resolveChangedNodeIDSet(
	baseNodes []models.Node,
	baseEdges []models.Edge,
	targetNodes []models.Node,
	targetEdges []models.Edge,
) map[string]struct{} {
	changed := make(map[string]struct{})

	baseByID := mapNodesByID(baseNodes)
	targetByID := mapNodesByID(targetNodes)
	allNodeIDs := make(map[string]struct{}, len(baseByID)+len(targetByID))
	for nodeID := range baseByID {
		allNodeIDs[nodeID] = struct{}{}
	}
	for nodeID := range targetByID {
		allNodeIDs[nodeID] = struct{}{}
	}

	for nodeID := range allNodeIDs {
		baseNode, hasBase := baseByID[nodeID]
		targetNode, hasTarget := targetByID[nodeID]
		if hasBase != hasTarget {
			changed[nodeID] = struct{}{}
			continue
		}
		if !hasBase {
			continue
		}
		if !reflect.DeepEqual(toComparableCanvasNode(baseNode), toComparableCanvasNode(targetNode)) {
			changed[nodeID] = struct{}{}
		}
	}

	baseEdgesByKey := mapEdgesByKey(baseEdges)
	targetEdgesByKey := mapEdgesByKey(targetEdges)
	for edgeKey, edge := range targetEdgesByKey {
		if _, ok := baseEdgesByKey[edgeKey]; ok {
			continue
		}
		if edge.SourceID != "" {
			changed[edge.SourceID] = struct{}{}
		}
		if edge.TargetID != "" {
			changed[edge.TargetID] = struct{}{}
		}
	}
	for edgeKey, edge := range baseEdgesByKey {
		if _, ok := targetEdgesByKey[edgeKey]; ok {
			continue
		}
		if edge.SourceID != "" {
			changed[edge.SourceID] = struct{}{}
		}
		if edge.TargetID != "" {
			changed[edge.TargetID] = struct{}{}
		}
	}

	return changed
}

func resolveOrderedNodeIDs(changedSet map[string]struct{}, orderedNodeGroups ...[]models.Node) []string {
	if len(changedSet) == 0 {
		return nil
	}

	result := make([]string, 0, len(changedSet))
	seen := make(map[string]struct{}, len(changedSet))

	for _, nodes := range orderedNodeGroups {
		for _, node := range nodes {
			if node.ID == "" {
				continue
			}
			if _, ok := changedSet[node.ID]; !ok {
				continue
			}
			if _, ok := seen[node.ID]; ok {
				continue
			}
			seen[node.ID] = struct{}{}
			result = append(result, node.ID)
		}
	}

	remaining := make([]string, 0, len(changedSet)-len(result))
	for nodeID := range changedSet {
		if _, ok := seen[nodeID]; ok {
			continue
		}
		remaining = append(remaining, nodeID)
	}
	sort.Strings(remaining)
	result = append(result, remaining...)

	return result
}

func mapNodesByID(nodes []models.Node) map[string]models.Node {
	result := make(map[string]models.Node, len(nodes))
	for _, node := range nodes {
		if node.ID == "" {
			continue
		}
		result[node.ID] = node
	}
	return result
}

func mapEdgesByKey(edges []models.Edge) map[string]models.Edge {
	result := make(map[string]models.Edge, len(edges))
	for _, edge := range edges {
		key := edge.SourceID + "|" + edge.TargetID + "|" + edge.Channel
		result[key] = edge
	}
	return result
}

func toComparableCanvasNode(node models.Node) comparableCanvasNode {
	return comparableCanvasNode{
		ID:            node.ID,
		Name:          node.Name,
		Type:          node.Type,
		Ref:           node.Ref,
		Configuration: node.Configuration,
		Position:      node.Position,
		IsCollapsed:   node.IsCollapsed,
		IntegrationID: node.IntegrationID,
	}
}
