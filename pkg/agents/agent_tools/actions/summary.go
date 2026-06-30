package actions

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

func summarizeCanvasVersion(canvas *models.Canvas, version *models.CanvasVersion) summary {
	summary := summary{}
	if canvas != nil {
		summary.CanvasName = canvas.Name
	}
	if version == nil {
		return summary
	}
	if summary.CanvasName == "" {
		summary.CanvasName = version.GitBranch
	}
	summary.NodeCount = len(version.Nodes)
	summary.EdgeCount = len(version.Edges)
	summary.Nodes = summarizeNodes(version.Nodes, 20)
	return summary
}

// summarizeParsedCanvas builds a summary from nodes/edges parsed out of staged
// canvas.yaml. Staging never materializes the version row, so the agent's reads
// and writes summarize the parsed staged graph instead of a persisted version.
func summarizeParsedCanvas(canvasName string, nodes []models.Node, edges []models.Edge) summary {
	return summary{
		CanvasName: canvasName,
		NodeCount:  len(nodes),
		EdgeCount:  len(edges),
		Nodes:      summarizeNodes(nodes, 20),
	}
}

func summarizeNodes(nodes []models.Node, limit int) []nodeSummary {
	count := len(nodes)
	if count > limit {
		count = limit
	}
	result := make([]nodeSummary, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, nodeSummary{
			ID:        nodes[i].ID,
			Name:      nodes[i].Name,
			Type:      nodes[i].Type,
			Component: nodeRefName(nodes[i].Ref),
			Issue:     firstNonEmptyString(stringPtrValue(nodes[i].ErrorMessage), stringPtrValue(nodes[i].WarningMessage)),
		})
	}
	return result
}

func collectNodeIssues(nodes []models.Node) []nodeIssue {
	issues := []nodeIssue{}
	for _, node := range nodes {
		if node.ErrorMessage != nil && strings.TrimSpace(*node.ErrorMessage) != "" {
			issues = append(issues, nodeIssue{NodeID: node.ID, NodeName: node.Name, Severity: "error", Message: strings.TrimSpace(*node.ErrorMessage)})
		}
		if node.WarningMessage != nil && strings.TrimSpace(*node.WarningMessage) != "" {
			issues = append(issues, nodeIssue{NodeID: node.ID, NodeName: node.Name, Severity: "warning", Message: strings.TrimSpace(*node.WarningMessage)})
		}
	}
	return issues
}

func nodeRefName(ref models.NodeRef) string {
	switch {
	case ref.Component != nil:
		return ref.Component.Name
	case ref.Trigger != nil:
		return ref.Trigger.Name
	case ref.Widget != nil:
		return ref.Widget.Name
	default:
		return ""
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
