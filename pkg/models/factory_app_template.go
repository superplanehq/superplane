package models

const (
	FactoryAppTemplateMetadataKey = "factoryTemplate"
	FactoryAppTemplateBacklogID   = "backlog"
	FactoryAppBacklogTriggerID    = "trigger"
)

// FactoryAppTemplateMetadata returns the persisted marker for a generated
// factory app. The marker survives canvas renames and user-facing copy changes.
func FactoryAppTemplateMetadata(templateID string, version int) map[string]any {
	return map[string]any{
		FactoryAppTemplateMetadataKey: map[string]any{
			"id":      templateID,
			"version": version,
		},
	}
}

func FactoryAppTemplateID(nodes []Node) string {
	for _, node := range nodes {
		metadata, ok := node.Metadata[FactoryAppTemplateMetadataKey].(map[string]any)
		if !ok {
			continue
		}
		if id, ok := metadata["id"].(string); ok {
			return id
		}
	}
	return ""
}

func IsBacklogFactoryApp(nodes []Node, edges []Edge) bool {
	if FactoryAppTemplateID(nodes) == FactoryAppTemplateBacklogID {
		return true
	}

	if len(nodes) != 5 || len(edges) != 4 {
		return false
	}
	requiredNodes := map[string]string{
		FactoryAppBacklogTriggerID: "onWorkOrder",
		"analyze":                  "",
		"report-confidence":        "reportWorkOrderCheck",
		"attach-intent":            "addWorkOrderArtifact",
		"add-run-error":            "addRunError",
	}
	analysisComponents := map[string]bool{
		SuperPlaneRunnerComponent: true,
		"runnerClaudeCode":        true,
		"runnerCodex":             true,
		"runnerOpenRouter":        true,
	}
	for _, node := range nodes {
		expectedComponent, ok := requiredNodes[node.ID]
		if !ok {
			return false
		}
		if node.ID == "analyze" {
			if !analysisComponents[node.ComponentName()] {
				return false
			}
			continue
		}
		if node.ComponentName() != expectedComponent {
			return false
		}
	}

	requiredEdges := map[Edge]bool{
		{SourceID: FactoryAppBacklogTriggerID, TargetID: "analyze", Channel: "default"}: true,
		{SourceID: "analyze", TargetID: "report-confidence", Channel: "passed"}:         true,
		{SourceID: "analyze", TargetID: "attach-intent", Channel: "passed"}:             true,
		{SourceID: "analyze", TargetID: "add-run-error", Channel: "failed"}:             true,
	}
	for _, edge := range edges {
		if !requiredEdges[edge] {
			return false
		}
	}
	return true
}
