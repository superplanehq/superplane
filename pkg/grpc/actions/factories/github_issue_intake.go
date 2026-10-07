package factories

import (
	"maps"
	"reflect"
	"slices"

	"github.com/superplanehq/superplane/pkg/models"
)

const intakeNodeSpacing = 180

const (
	intakeConfigLabels               = "labels"
	intakeConfigLabelFilterMode      = "labelFilterMode"
	intakeConfigAssignment           = "assignment"
	intakeConfigAuthorsWithAccess    = "authorsWithAccess"
	intakeConfigSuperplaneLabelAdded = "superplaneLabelAdded"
)

var intakeGitHubFilterNodeIDs = []string{
	intakeFilterNodeID,
	intakeAuthorPermissionNodeID,
	intakeAuthorFilterNodeID,
}

func applyGitHubIssueFilterConfiguration(configuration map[string]any, settings intakeSettings) {
	settings = settings.normalized()
	configuration["actions"] = intakeTriggerActionsFor(settings)
	configuration[intakeConfigLabels] = configurationAnyStrings(settings.Labels)
	setOrDeleteConfiguration(configuration, intakeConfigLabelFilterMode, settings.LabelFilterMode == intakeLabelFilterExclude, intakeLabelFilterExclude)
	setOrDeleteConfiguration(configuration, intakeConfigAssignment, settings.Assignment == intakeAssignmentAssigned || settings.Assignment == intakeAssignmentUnassigned, settings.Assignment)
	setOrDeleteConfiguration(configuration, intakeConfigAuthorsWithAccess, settings.AuthorsWithAccess, true)
	setOrDeleteConfiguration(configuration, intakeConfigSuperplaneLabelAdded, settings.SuperplaneLabelAdded, true)
}

func setOrDeleteConfiguration(configuration map[string]any, key string, set bool, value any) {
	if set {
		configuration[key] = value
		return
	}
	delete(configuration, key)
}

func githubIssueFiltersConfigured(configuration map[string]any) bool {
	if configuration == nil {
		return false
	}
	for _, key := range []string{
		intakeConfigLabels,
		intakeConfigLabelFilterMode,
		intakeConfigAssignment,
		intakeConfigAuthorsWithAccess,
		intakeConfigSuperplaneLabelAdded,
	} {
		if _, ok := configuration[key]; ok {
			return true
		}
	}
	return false
}

func githubIssueFiltersMigrated(nodes []models.Node) bool {
	for _, id := range intakeGitHubFilterNodeIDs {
		if findIntakeNode(nodes, id) != nil {
			return false
		}
	}
	return true
}

func githubIssueFiltersFromTrigger(configuration map[string]any, settings intakeSettings) intakeSettings {
	if labels, ok := configuration[intakeConfigLabels]; ok {
		settings.Labels = configurationStrings(labels)
	}
	if _, ok := configuration[intakeConfigLabelFilterMode]; ok {
		settings.LabelFilterMode = intakeLabelFilterInclude
		if mode, _ := configuration[intakeConfigLabelFilterMode].(string); mode == intakeLabelFilterExclude {
			settings.LabelFilterMode = intakeLabelFilterExclude
		}
	}
	if _, ok := configuration[intakeConfigAssignment]; ok {
		settings.Assignment = intakeAssignmentAny
		switch assignment, _ := configuration[intakeConfigAssignment].(string); assignment {
		case intakeAssignmentAssigned, intakeAssignmentUnassigned:
			settings.Assignment = assignment
		}
	}
	if _, ok := configuration[intakeConfigAuthorsWithAccess]; ok {
		settings.AuthorsWithAccess = metadataBool(configuration[intakeConfigAuthorsWithAccess], false)
	}
	if _, ok := configuration[intakeConfigSuperplaneLabelAdded]; ok {
		settings.SuperplaneLabelAdded = metadataBool(configuration[intakeConfigSuperplaneLabelAdded], false)
	}
	return settings
}

func writeGitHubIssueFilters(nodes []models.Node, triggerID string, settings intakeSettings) []models.Node {
	if triggerID == "" {
		return nodes
	}
	for i := range nodes {
		if nodes[i].ID != triggerID {
			continue
		}
		configuration := maps.Clone(nodes[i].Configuration)
		if configuration == nil {
			configuration = map[string]any{}
		}
		applyGitHubIssueFilterConfiguration(configuration, settings)
		nodes[i].Configuration = configuration
	}
	return nodes
}

func collapseGitHubIntakeGraph(nodes []models.Node, edges []models.Edge, graph intakeGraph) ([]models.Node, []models.Edge) {
	removed := make(map[string]bool, len(intakeGitHubFilterNodeIDs))
	for _, id := range intakeGitHubFilterNodeIDs {
		removed[id] = true
	}
	bypassed := intakeRouteEdges(edges, graph.TriggerNodeID, graph.CreateNodeID)

	nodes = slices.DeleteFunc(slices.Clone(nodes), func(node models.Node) bool {
		return removed[node.ID]
	})
	edges = slices.DeleteFunc(slices.Clone(edges), func(edge models.Edge) bool {
		if removed[edge.SourceID] || removed[edge.TargetID] {
			return true
		}
		_, onRoute := bypassed[edge]
		return onRoute
	})

	if graph.TriggerNodeID == "" || graph.CreateNodeID == "" {
		return nodes, edges
	}

	edges = slices.DeleteFunc(edges, func(edge models.Edge) bool {
		return edge.SourceID == graph.TriggerNodeID && edge.TargetID == graph.CreateNodeID
	})
	edges = ensureIntakeEdge(edges, models.Edge{
		Channel:  "default",
		SourceID: graph.TriggerNodeID,
		TargetID: graph.CreateNodeID,
	})
	placeCreateUnderTrigger(nodes, graph.TriggerNodeID, graph.CreateNodeID)
	return nodes, edges
}

func placeCreateUnderTrigger(nodes []models.Node, triggerID, createID string) {
	trigger := findIntakeNode(nodes, triggerID)
	if trigger == nil {
		return
	}
	for i := range nodes {
		if nodes[i].ID != createID {
			continue
		}
		nodes[i].Position = models.Position{X: trigger.Position.X, Y: trigger.Position.Y + intakeNodeSpacing}
	}
}

func intakeRouteEdges(edges []models.Edge, sourceID, targetID string) map[models.Edge]struct{} {
	found := map[models.Edge]struct{}{}
	if sourceID == "" || targetID == "" || sourceID == targetID {
		return found
	}

	var walk func(current string, visited map[string]bool) bool
	walk = func(current string, visited map[string]bool) bool {
		reaches := false
		for _, edge := range edges {
			if edge.SourceID != current || visited[edge.TargetID] {
				continue
			}
			if edge.TargetID == targetID {
				found[edge] = struct{}{}
				reaches = true
				continue
			}
			next := maps.Clone(visited)
			next[edge.TargetID] = true
			if walk(edge.TargetID, next) {
				found[edge] = struct{}{}
				reaches = true
			}
		}
		return reaches
	}
	walk(sourceID, map[string]bool{sourceID: true})
	return found
}

func rewriteGitHubIntakeGraph(
	nodes []models.Node,
	edges []models.Edge,
	graph intakeGraph,
	settings intakeSettings,
) ([]models.Node, []models.Edge, bool) {
	updatedNodes := writeGitHubIssueFilters(slices.Clone(nodes), graph.TriggerNodeID, settings)
	updatedNodes, updatedEdges := collapseGitHubIntakeGraph(updatedNodes, edges, graph)
	changed := !reflect.DeepEqual(nodes, updatedNodes) || !reflect.DeepEqual(edges, updatedEdges)
	return updatedNodes, updatedEdges, changed
}
