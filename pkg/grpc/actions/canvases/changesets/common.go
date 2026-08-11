package changesets

import (
	"fmt"
	"sort"
	"strings"

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

	// Kahn's never enqueues a node twice, so tracking which nodes were sorted
	// leaves exactly the nodes involved in (or downstream of) a cycle behind.
	sorted := make(map[string]struct{}, len(inDegree))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted[current] = struct{}{}

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If we visited all nodes, the graph is acyclic
	if len(sorted) != len(nodes) {
		remaining := make(map[string]struct{})
		for _, node := range nodes {
			if _, ok := sorted[node.ID]; !ok {
				remaining[node.ID] = struct{}{}
			}
		}

		if cycle := findCycle(graph, remaining); len(cycle) > 0 {
			return fmt.Errorf("graph contains a cycle: %s", describeCycle(nodes, cycle))
		}

		return fmt.Errorf("graph contains a cycle")
	}

	return nil
}

/*
 * findCycle returns one cycle as an ordered list of node IDs with the entry node
 * repeated at the end, so the caller can render it as "A -> B -> A". Only nodes
 * in candidates are traversed, which is the set Kahn's algorithm could not sort.
 */
func findCycle(graph map[string][]string, candidates map[string]struct{}) []string {
	const (
		unvisited = iota
		onStack
		done
	)

	state := make(map[string]int, len(candidates))
	stack := []string{}
	var cycle []string

	var visit func(string) bool
	visit = func(id string) bool {
		state[id] = onStack
		stack = append(stack, id)

		for _, next := range graph[id] {
			if _, ok := candidates[next]; !ok {
				continue
			}

			if state[next] == onStack {
				for i, node := range stack {
					if node == next {
						cycle = append(append([]string{}, stack[i:]...), next)
						break
					}
				}

				return true
			}

			if state[next] == unvisited && visit(next) {
				return true
			}
		}

		stack = stack[:len(stack)-1]
		state[id] = done
		return false
	}

	// Sort the entry points so the reported cycle is stable across runs.
	entries := make([]string, 0, len(candidates))
	for id := range candidates {
		entries = append(entries, id)
	}
	sort.Strings(entries)

	for _, id := range entries {
		if state[id] == unvisited && visit(id) {
			return cycle
		}
	}

	return nil
}

func describeCycle(nodes []models.Node, cycle []string) string {
	names := make(map[string]string, len(nodes))
	for _, node := range nodes {
		if node.Name != "" {
			names[node.ID] = node.Name
		}
	}

	labels := make([]string, 0, len(cycle))
	for _, id := range cycle {
		if name, ok := names[id]; ok {
			labels = append(labels, name)
			continue
		}

		labels = append(labels, id)
	}

	return strings.Join(labels, " -> ")
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
