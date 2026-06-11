package changesets

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
)

// Verify if the workflow is acyclic using
// topological sort algorithm - kahn's - to detect cycles
func CheckForCycles(nodes []models.Node, edges []models.Edge) error {
	loopNodeIDs := loopNodeIDSet(nodes)

	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, node := range nodes {
		graph[node.ID] = []string{}
		inDegree[node.ID] = 0
	}

	for _, edge := range edges {
		if _, isLoopNode := loopNodeIDs[edge.TargetID]; isLoopNode {
			continue
		}

		graph[edge.SourceID] = append(graph[edge.SourceID], edge.TargetID)
		inDegree[edge.TargetID]++
	}

	// Kahn's algorithm for topological sort
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we visited all nodes, the graph is acyclic
	if visited != len(nodes) {
		return fmt.Errorf("graph contains a cycle")
	}

	return nil
}

func loopNodeIDSet(nodes []models.Node) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, node := range nodes {
		if node.Ref.Component != nil && node.Ref.Component.Name == "loop" {
			ids[node.ID] = struct{}{}
		}
	}
	return ids
}
