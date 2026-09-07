package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func Test__BuildIntakeCanvas(t *testing.T) {
	t.Run("each source listens with its own trigger", func(t *testing.T) {
		for source, expected := range map[string]string{
			models.FactoryIntakeSourceGitHubIssues:       "github.onIssue",
			models.FactoryIntakeSourceSentryExceptions:   "sentry.onIssue",
			models.FactoryIntakeSourcePagerDutyIncidents: "pagerduty.onIncident",
			models.FactoryIntakeSourceProductiveTasks:    "productive.onTask",
		} {
			canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: source})
			require.NoError(t, err)

			trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
			assert.Equal(t, expected, trigger.Component)
			assert.Equal(t, yaml.NodeTypeTrigger, trigger.Type)
		}
	})

	t.Run("a GitHub issue flows from the trigger through the filter to the work order", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues})
		require.NoError(t, err)

		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, canvas.Spec.Edges)
		assert.Nil(t, findSpecNodeOrNil(canvas, intakeAnalysisNodeID))
		assert.Nil(t, findSpecNodeOrNil(canvas, intakeReportConfidenceNodeID))

		filter := findSpecNode(t, canvas, intakeFilterNodeID)
		assert.Equal(t, intakeFilterComponent, filter.Component)
		assert.Equal(t, "true", filter.Configuration["expression"])
	})

	t.Run("Sentry, PagerDuty, and Productive.io create a work order without a filter", func(t *testing.T) {
		for _, source := range []string{
			models.FactoryIntakeSourceSentryExceptions,
			models.FactoryIntakeSourcePagerDutyIncidents,
			models.FactoryIntakeSourceProductiveTasks,
		} {
			canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: source})
			require.NoError(t, err)
			assert.Equal(t, []yaml.Edge{
				{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID},
			}, canvas.Spec.Edges)
			assert.Nil(t, findSpecNodeOrNil(canvas, intakeFilterNodeID))
		}
	})

	t.Run("every action node works on a whole batch at once", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues})
		require.NoError(t, err)

		for _, node := range canvas.Spec.Nodes {
			if node.Type == yaml.NodeTypeTrigger {
				assert.Nilf(t, node.Concurrency, "trigger %s caps its concurrency", node.ID)
				continue
			}

			require.NotNilf(t, node.Concurrency, "node %s has no concurrency", node.ID)
			require.NotNilf(t, node.Concurrency.Max, "node %s has no concurrency max", node.ID)
			assert.Equalf(t, intakeConcurrencyMax, *node.Concurrency.Max, "node %s", node.ID)
		}
	})

	t.Run("a given name wins over the source default", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Name: "Backlog triage"})
		require.NoError(t, err)
		assert.Equal(t, "Backlog triage", canvas.Metadata.Name)

		canvas, err = buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Name: "   "})
		require.NoError(t, err)
		assert.Equal(t, "GitHub issues", canvas.Metadata.Name)
	})

	t.Run("an unknown source has no graph", func(t *testing.T) {
		_, err := buildIntakeCanvas(intakeCanvasRequest{Source: "linear-issues"})
		assert.ErrorIs(t, err, models.ErrFactoryIntakeSourceInvalid)
	})

	t.Run("the binding tells the trigger what to listen on", func(t *testing.T) {
		binding := &intakeBinding{
			Integration:   &yaml.IntegrationRef{ID: "integration-1", Name: "acme-github"},
			Configuration: map[string]any{"repository": "acme/backlog"},
		}
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{
			Source:  models.FactoryIntakeSourceGitHubIssues,
			Binding: binding,
		})
		require.NoError(t, err)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Equal(t, binding.Integration, trigger.Integration)
		assert.Equal(t, "acme/backlog", trigger.Configuration["repository"])
		assert.Equal(t, []any{"opened"}, trigger.Configuration["actions"])
	})

	t.Run("a binding does not leak into the next intake", func(t *testing.T) {
		_, err := buildIntakeCanvas(intakeCanvasRequest{
			Source: models.FactoryIntakeSourceGitHubIssues,
			Binding: &intakeBinding{
				Configuration: map[string]any{"repository": "acme/backlog"},
			},
		})
		require.NoError(t, err)

		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues})
		require.NoError(t, err)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Nil(t, trigger.Integration)
		assert.NotContains(t, trigger.Configuration, "repository")
	})
}

func Test__IntakeFilterExpression(t *testing.T) {
	t.Run("an empty GitHub filter is true", func(t *testing.T) {
		assert.Equal(t, "true", intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, defaultIntakeSettings()))
	})

	t.Run("GitHub labels and assignment join without a score", func(t *testing.T) {
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, intakeSettings{
			Labels:          []string{"bug"},
			LabelFilterMode: intakeLabelFilterExclude,
			Assignment:      intakeAssignmentUnassigned,
		})
		assert.NotContains(t, expression, ">=")
		assert.Contains(t, expression, `!(root().data.issue.labels.exists(label, label.name in ["bug"]))`)
		assert.Contains(t, expression, intakeUnassignedCondition)
	})
}

func findSpecNode(t *testing.T, canvas *yaml.Canvas, nodeID string) yaml.Node {
	t.Helper()

	node := findSpecNodeOrNil(canvas, nodeID)
	if node == nil {
		require.Failf(t, "node not found", "canvas has no node %q", nodeID)
		return yaml.Node{}
	}
	return *node
}

func findSpecNodeOrNil(canvas *yaml.Canvas, nodeID string) *yaml.Node {
	if canvas == nil || canvas.Spec == nil {
		return nil
	}
	for i := range canvas.Spec.Nodes {
		if canvas.Spec.Nodes[i].ID == nodeID {
			return &canvas.Spec.Nodes[i]
		}
	}
	return nil
}
