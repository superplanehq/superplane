package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestDetectFactoryIntake(t *testing.T) {
	tests := []struct {
		name          string
		componentName string
		source        factoryIntakeSource
	}{
		{
			name:          "GitHub issues",
			componentName: "github.onIssue",
			source:        factoryIntakeSourceGitHubIssues,
		},
		{
			name:          "Sentry exceptions",
			componentName: "sentry.onIssue",
			source:        factoryIntakeSourceSentryExceptions,
		},
		{
			name:          "PagerDuty incidents",
			componentName: "pagerduty.onIncident",
			source:        factoryIntakeSourcePagerDutyIncidents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := &models.CanvasVersion{
				Nodes: []models.Node{
					triggerNode("intake-trigger", tt.componentName),
					actionNode("analyze-intake", "runnerClaudeCode"),
					actionNode("create-work-order", "createWorkOrder"),
				},
				Edges: []models.Edge{
					{SourceID: "intake-trigger", TargetID: "analyze-intake", Channel: "default"},
					{SourceID: "analyze-intake", TargetID: "create-work-order", Channel: "passed"},
				},
			}

			intake, ok := detectFactoryIntake(version)

			assert.True(t, ok)
			assert.Equal(t, tt.source, intake.Source)
			assert.Equal(t, "intake-trigger", intake.TriggerNodeID)
			assert.Equal(t, "analyze-intake", intake.AnalysisNodeID)
			assert.Equal(t, "create-work-order", intake.CreateWorkOrderNodeID)
		})
	}
}

func TestDetectFactoryIntakeRejectsUnrelatedAutomation(t *testing.T) {
	version := &models.CanvasVersion{
		Nodes: []models.Node{
			triggerNode("trigger", "github.onIssue"),
			actionNode("analysis", "runnerClaudeCode"),
			actionNode("create", "createWorkOrder"),
		},
		Edges: []models.Edge{{SourceID: "trigger", TargetID: "analysis", Channel: "default"}},
	}

	_, ok := detectFactoryIntake(version)

	assert.False(t, ok)
}

func TestSerializeFactoryAppsIncludesIntakeMetadata(t *testing.T) {
	canvasID := uuid.New()
	apps := serializeFactoryApps(
		[]models.Canvas{{ID: canvasID, Name: "GitHub issues"}},
		map[uuid.UUID]factoryIntake{
			canvasID: {
				Source:                factoryIntakeSourceGitHubIssues,
				TriggerNodeID:         "trigger",
				AnalysisNodeID:        "analysis",
				CreateWorkOrderNodeID: "create",
			},
		},
	)

	assert.Equal(t, pb.Factory_App_Intake_SOURCE_GITHUB_ISSUES, apps[0].GetIntake().GetSource())
	assert.Equal(t, "trigger", apps[0].GetIntake().GetTriggerNodeId())
	assert.Equal(t, "analysis", apps[0].GetIntake().GetAnalysisNodeId())
	assert.Equal(t, "create", apps[0].GetIntake().GetCreateWorkOrderNodeId())
}

func triggerNode(id, componentName string) models.Node {
	return models.Node{
		ID:   id,
		Type: "TYPE_TRIGGER",
		Ref: models.NodeRef{
			Trigger: &models.TriggerRef{Name: componentName},
		},
	}
}

func actionNode(id, componentName string) models.Node {
	return models.Node{
		ID:   id,
		Type: "TYPE_ACTION",
		Ref: models.NodeRef{
			Component: &models.ComponentRef{Name: componentName},
		},
	}
}
