package canvases

import "github.com/superplanehq/superplane/pkg/models"

const groupWidgetName = "group"

func isGroupWidgetNode(node models.Node) bool {
	return node.Type == models.NodeTypeWidget &&
		node.Ref.Widget != nil &&
		node.Ref.Widget.Name == groupWidgetName
}

func getGroupChildNodeIDs(node models.Node) []string {
	if node.Configuration == nil {
		return nil
	}

	childNodeIDs, ok := node.Configuration["childNodeIds"]
	if !ok {
		return nil
	}

	switch typed := childNodeIDs.(type) {
	case []string:
		result := make([]string, 0, len(typed))
		for _, childID := range typed {
			if childID != "" {
				result = append(result, childID)
			}
		}
		return result
	case []any:
		result := make([]string, 0, len(typed))
		for _, value := range typed {
			childID, ok := value.(string)
			if ok && childID != "" {
				result = append(result, childID)
			}
		}
		return result
	default:
		return nil
	}
}

func normalizeCanvasNodesWithoutGroups(nodes []models.Node) []models.Node {
	if len(nodes) == 0 {
		return nodes
	}

	hasGroups := false
	groupNodesByID := make(map[string]models.Node)
	parentGroupByChildID := make(map[string]string)

	for _, node := range nodes {
		if !isGroupWidgetNode(node) {
			continue
		}

		hasGroups = true
		if node.ID == "" {
			continue
		}

		groupNodesByID[node.ID] = node
		for _, childID := range getGroupChildNodeIDs(node) {
			if _, exists := parentGroupByChildID[childID]; !exists {
				parentGroupByChildID[childID] = node.ID
			}
		}
	}

	if !hasGroups {
		return nodes
	}

	offsetsByNodeID := make(map[string]models.Position)
	for _, node := range nodes {
		if node.ID == "" || isGroupWidgetNode(node) {
			continue
		}

		offset := models.Position{}
		currentGroupID := parentGroupByChildID[node.ID]
		visited := make(map[string]bool)

		for currentGroupID != "" && !visited[currentGroupID] {
			visited[currentGroupID] = true
			groupNode, ok := groupNodesByID[currentGroupID]
			if !ok {
				break
			}

			offset.X += groupNode.Position.X
			offset.Y += groupNode.Position.Y
			currentGroupID = parentGroupByChildID[currentGroupID]
		}

		if offset.X != 0 || offset.Y != 0 {
			offsetsByNodeID[node.ID] = offset
		}
	}

	normalized := make([]models.Node, 0, len(nodes))
	for _, node := range nodes {
		if isGroupWidgetNode(node) {
			continue
		}

		offset, ok := offsetsByNodeID[node.ID]
		if ok {
			node.Position.X += offset.X
			node.Position.Y += offset.Y
		}

		normalized = append(normalized, node)
	}

	return normalized
}
