package factories

import "github.com/superplanehq/superplane/pkg/models"

type factoryIntakeSource string

const (
	factoryIntakeSourceGitHubIssues       factoryIntakeSource = "github-issues"
	factoryIntakeSourceSentryExceptions   factoryIntakeSource = "sentry-exceptions"
	factoryIntakeSourcePagerDutyIncidents factoryIntakeSource = "pagerduty-incidents"
)

type factoryIntake struct {
	Source                factoryIntakeSource
	TriggerNodeID         string
	AnalysisNodeID        string
	CreateWorkOrderNodeID string
}

var factoryIntakeSourceByTrigger = map[string]factoryIntakeSource{
	"github.onIssue":       factoryIntakeSourceGitHubIssues,
	"sentry.onIssue":       factoryIntakeSourceSentryExceptions,
	"pagerduty.onIncident": factoryIntakeSourcePagerDutyIncidents,
}

func detectFactoryIntake(version *models.CanvasVersion) (factoryIntake, bool) {
	if version == nil {
		return factoryIntake{}, false
	}

	var intake factoryIntake
	for i := range version.Nodes {
		node := &version.Nodes[i]
		componentName := node.ComponentName()

		if source, ok := factoryIntakeSourceByTrigger[componentName]; ok {
			intake.Source = source
			intake.TriggerNodeID = node.ID
		}
		if componentName == "runnerClaudeCode" {
			intake.AnalysisNodeID = node.ID
		}
		if componentName == "createWorkOrder" {
			intake.CreateWorkOrderNodeID = node.ID
		}
	}

	complete := intake.Source != "" &&
		intake.TriggerNodeID != "" &&
		intake.AnalysisNodeID != "" &&
		intake.CreateWorkOrderNodeID != "" &&
		hasCanvasPath(version.Edges, intake.TriggerNodeID, intake.AnalysisNodeID) &&
		hasCanvasPath(version.Edges, intake.AnalysisNodeID, intake.CreateWorkOrderNodeID)
	return intake, complete
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
