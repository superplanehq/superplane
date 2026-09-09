package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__intakeFilterExpressionFor_AuthorsWithAccess(t *testing.T) {
	t.Run("off by default", func(t *testing.T) {
		settings := defaultIntakeSettings()
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)
		assert.Equal(t, "true", expression)
	})

	t.Run("appends the author access condition when on", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.AuthorsWithAccess = true

		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)
		assert.Contains(t, expression, intakeAuthorAccessCondition)
	})

	t.Run("ignored for a source other than GitHub issues", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.AuthorsWithAccess = true

		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceSentryExceptions, settings)
		assert.Equal(t, "true", expression)
	})
}

func Test__intakeSettingsFromGraph_AuthorsWithAccess(t *testing.T) {
	newSpec := func(expression string) models.LiveCanvasSpec {
		return models.LiveCanvasSpec{
			Nodes: []models.Node{
				{
					ID: "filter",
					Configuration: map[string]any{
						"expression": expression,
					},
				},
			},
		}
	}

	t.Run("round-trips true through build and parse", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.AuthorsWithAccess = true
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		graph := intakeGraph{FilterNodeID: "filter"}
		parsed := intakeSettingsFromGraph(graph, newSpec(expression))

		assert.True(t, parsed.AuthorsWithAccess)
	})

	t.Run("reads false when the condition is absent", func(t *testing.T) {
		settings := defaultIntakeSettings()
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		graph := intakeGraph{FilterNodeID: "filter"}
		parsed := intakeSettingsFromGraph(graph, newSpec(expression))

		assert.False(t, parsed.AuthorsWithAccess)
	})

	t.Run("a hand-edited expression falls back to the default", func(t *testing.T) {
		graph := intakeGraph{FilterNodeID: "filter"}
		parsed := intakeSettingsFromGraph(graph, newSpec(`root().data.issue.author_association == "OWNER"`))

		assert.False(t, parsed.AuthorsWithAccess)
	})
}

func Test__intakeTriggerActionsFor(t *testing.T) {
	t.Run("listens for opened and reopened issues", func(t *testing.T) {
		settings := defaultIntakeSettings()

		assert.Equal(t, []any{"opened", "reopened"}, intakeTriggerActionsFor(settings))
	})

	t.Run("also listens for assignments", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.AssignedToAgent = true

		assert.Equal(t, []any{"opened", "reopened", "assigned"}, intakeTriggerActionsFor(settings))
	})

	t.Run("can listen only for assignments", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.NewIssues = false
		settings.AssignedToAgent = true

		assert.Equal(t, []any{"assigned"}, intakeTriggerActionsFor(settings))
	})
}

func Test__intakeSettingsFromGraph_TriggerActions(t *testing.T) {
	spec := models.LiveCanvasSpec{
		Nodes: []models.Node{
			{
				ID: intakeTriggerNodeID,
				Configuration: map[string]any{
					"actions": []any{"assigned"},
				},
			},
		},
	}

	settings := intakeSettingsFromGraph(intakeGraph{TriggerNodeID: intakeTriggerNodeID}, spec)

	assert.False(t, settings.NewIssues)
	assert.True(t, settings.AssignedToAgent)
}

func Test__intakeFilterExpressionFor_AssignedToAgent(t *testing.T) {
	settings := defaultIntakeSettings()
	settings.AssignedToAgent = true

	expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

	assert.Contains(t, expression, intakeAssignedToAgentCondition)
}
