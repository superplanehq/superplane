package factories

import (
	"slices"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Keep the installed trigger identity when a template rebuild changes settings.
// Its runtime metadata and provider resources belong to that live node.
func alignConfigurationIntakeTrigger(current *models.CanvasVersion, source string, nodes []models.Node, edges []models.Edge) ([]models.Node, []models.Edge) {
	if current == nil || source == "" {
		return nodes, edges
	}
	installed := resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: current.Nodes, Edges: current.Edges})
	desired := resolveIntakeGraph(source, models.LiveCanvasSpec{Nodes: nodes, Edges: edges})
	return renameConfigurationIntakeTrigger(nodes, edges, desired.TriggerNodeID, installed.TriggerNodeID)
}

// Binding changes retire provider state through the publisher's normal node
// deletion path. In particular, Datadog cleanup needs the deleted node's old
// integration and monitor metadata after the transaction commits.
func replaceConfigurationIntakeTriggerForBinding(tx *gorm.DB, current *factoryConfigurationAutomationTarget, prepared *preparedFactoryConfigurationBuiltIn, nodes []models.Node, edges []models.Edge) ([]models.Node, []models.Edge, error) {
	if current == nil || prepared.IntakeSource == "" || current.binding == prepared.Binding {
		return nodes, edges, nil
	}
	graph := resolveIntakeGraph(prepared.IntakeSource, models.LiveCanvasSpec{Nodes: nodes, Edges: edges})
	trigger := findIntakeNode(nodes, graph.TriggerNodeID)
	if trigger == nil {
		return nil, nil, invalidArgument("The intake automation has no trigger.")
	}
	previous, err := models.FindCanvasNodesUnscopedInTransaction(tx, current.canvas.ID)
	if err != nil {
		return nil, nil, err
	}
	reserved := make(map[string]bool, len(previous)+len(nodes))
	for _, node := range previous {
		reserved[node.NodeID] = true
	}
	for _, node := range nodes {
		reserved[node.ID] = true
	}
	nodes, edges = renameConfigurationIntakeTrigger(nodes, edges, trigger.ID, models.GenerateUniqueNodeID(*trigger, reserved))
	return nodes, edges, nil
}

func renameConfigurationIntakeTrigger(nodes []models.Node, edges []models.Edge, previous, current string) ([]models.Node, []models.Edge) {
	if previous == "" || current == "" || previous == current {
		return nodes, edges
	}
	nodes = slices.Clone(nodes)
	for i := range nodes {
		if nodes[i].ID == previous {
			nodes[i].ID = current
		}
	}
	edges = slices.Clone(edges)
	for i := range edges {
		if edges[i].SourceID == previous {
			edges[i].SourceID = current
		}
		if edges[i].TargetID == previous {
			edges[i].TargetID = current
		}
	}
	return nodes, edges
}
