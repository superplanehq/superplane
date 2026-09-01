package factories

import (
	"slices"

	"github.com/superplanehq/superplane/pkg/models"
)

// intakeGraph locates the nodes of an intake inside its canvas. Node
// identifiers are an implementation detail of the generated graph, so they are
// resolved on read and never stored on the intake row or sent over the API.
type intakeGraph struct {
	TriggerNodeID  string
	AnalysisNodeID string
	FilterNodeID   string
	CreateNodeID   string
	ConfidencePct  int
}

// Healthy reports whether the graph can still do the intake's job: receive an
// item and reach the node that creates the work order. Analysis is optional:
// a generated graph creates first, and a legacy graph that still scores stays
// healthy as long as create is reachable.
func (g intakeGraph) Healthy(edges []models.Edge) bool {
	if g.TriggerNodeID == "" || g.CreateNodeID == "" {
		return false
	}

	return hasCanvasPath(edges, g.TriggerNodeID, g.CreateNodeID)
}

// resolveIntakeGraph matches the generated node identifiers first, then falls
// back to component names so a renamed node or a different agent runner does
// not make the intake unreadable.
func resolveIntakeGraph(source string, spec models.LiveCanvasSpec) intakeGraph {
	nodes := spec.Nodes
	graph := intakeGraph{
		ConfidencePct: DefaultIntakeConfidencePct,
	}

	triggerComponent := intakeSpecsBySource[source].triggerComponent
	graph.TriggerNodeID = resolveIntakeNode(nodes, intakeTriggerNodeID, func(node *models.Node) bool {
		return node.ComponentName() == triggerComponent
	})
	graph.AnalysisNodeID = resolveIntakeNode(nodes, intakeAnalysisNodeID, func(node *models.Node) bool {
		return slices.Contains(intakeAnalysisComponents, node.ComponentName())
	})
	graph.FilterNodeID = resolveIntakeFilterNode(nodes)
	graph.CreateNodeID = resolveIntakeNode(nodes, intakeCreateNodeID, func(node *models.Node) bool {
		return node.ComponentName() == intakeCreateComponent
	})

	if filter := findIntakeNode(nodes, graph.FilterNodeID); filter != nil {
		if expression, ok := filter.Configuration["expression"].(string); ok {
			if confidence, ok := intakeConfidenceFromExpression(expression); ok {
				graph.ConfidencePct = confidence
			}
		}
	}

	return graph
}

func resolveIntakeFilterNode(nodes []models.Node) string {
	if id := resolveIntakeNode(nodes, intakeFilterNodeID, func(node *models.Node) bool {
		return node.ComponentName() == intakeFilterComponent
	}); id != "" {
		return id
	}

	return resolveIntakeNode(nodes, intakeThresholdNodeID, func(node *models.Node) bool {
		return node.ComponentName() == intakeFilterComponent
	})
}

func resolveIntakeNode(nodes []models.Node, preferredID string, matches func(*models.Node) bool) string {
	if node := findIntakeNode(nodes, preferredID); node != nil {
		return node.ID
	}

	for i := range nodes {
		if matches(&nodes[i]) {
			return nodes[i].ID
		}
	}

	return ""
}

func findIntakeNode(nodes []models.Node, nodeID string) *models.Node {
	if nodeID == "" {
		return nil
	}

	for i := range nodes {
		if nodes[i].ID == nodeID {
			return &nodes[i]
		}
	}

	return nil
}

func hasCanvasPath(edges []models.Edge, sourceID, targetID string) bool {
	pending := []string{sourceID}
	visited := map[string]bool{sourceID: true}

	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		for _, edge := range edges {
			if edge.SourceID != current {
				continue
			}
			if edge.TargetID == targetID {
				return true
			}
			if !visited[edge.TargetID] {
				visited[edge.TargetID] = true
				pending = append(pending, edge.TargetID)
			}
		}
	}

	return false
}
