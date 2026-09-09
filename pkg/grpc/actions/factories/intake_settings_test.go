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

// The filter node runs the expression through expr-lang, so building it is not
// enough: it has to evaluate against the payloads GitHub actually sends. An
// `opened` payload carries no assignee, which is why every condition that
// reads one must sit behind a short-circuit.
func Test__intakeFilterExpressionFor_EvaluatesAgainstIssuePayloads(t *testing.T) {
	openedWithLabel := map[string]any{
		"action": "opened",
		"issue": map[string]any{
			"state":              "open",
			"author_association": "MEMBER",
			"assignees":          []any{},
			"labels":             []any{map[string]any{"name": "documentation"}},
		},
	}
	openedWithoutLabel := map[string]any{
		"action": "opened",
		"issue": map[string]any{
			"state":              "open",
			"author_association": "MEMBER",
			"assignees":          []any{},
			"labels":             []any{map[string]any{"name": "chore"}},
		},
	}
	assignedToAgent := map[string]any{
		"action":   "assigned",
		"assignee": map[string]any{"login": "superplaneagent"},
		"issue": map[string]any{
			"state":              "open",
			"author_association": "MEMBER",
			"assignees":          []any{map[string]any{"login": "superplaneagent"}},
			"labels":             []any{map[string]any{"name": "documentation"}},
		},
	}
	assignedToSomebodyElse := map[string]any{
		"action":   "assigned",
		"assignee": map[string]any{"login": "octocat"},
		"issue": map[string]any{
			"state":              "open",
			"author_association": "MEMBER",
			"assignees":          []any{map[string]any{"login": "octocat"}},
			"labels":             []any{map[string]any{"name": "documentation"}},
		},
	}

	t.Run("matches an issue that carries one of the labels", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation", "bug"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithLabel))
		assert.Equal(t, false, evalRootDataExpression(t, expression, openedWithoutLabel))
	})

	t.Run("excludes an issue that carries one of the labels", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation"}
		settings.LabelFilterMode = intakeLabelFilterExclude
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, false, evalRootDataExpression(t, expression, openedWithLabel))
		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithoutLabel))
	})

	t.Run("keeps the label filter for assignment events", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation"}
		settings.AssignedToAgent = true
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, assignedToAgent))
		assert.Equal(t, false, evalRootDataExpression(t, expression, assignedToSomebodyElse))
		// An opened issue has no assignee, so the agent check must not run.
		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithLabel))
		assert.Equal(t, false, evalRootDataExpression(t, expression, openedWithoutLabel))
	})

	t.Run("combines every filter", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation"}
		settings.Assignment = intakeAssignmentUnassigned
		settings.AuthorsWithAccess = true
		settings.AssignedToAgent = true
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithLabel))
		assert.Equal(t, false, evalRootDataExpression(t, expression, assignedToAgent))
	})
}

func Test__intakeSettingsFromGraph_Labels(t *testing.T) {
	newSpec := func(expression string) models.LiveCanvasSpec {
		return models.LiveCanvasSpec{
			Nodes: []models.Node{
				{
					ID:            "filter",
					Configuration: map[string]any{"expression": expression},
				},
			},
		}
	}
	graph := intakeGraph{FilterNodeID: "filter"}

	t.Run("round-trips the labels through build and parse", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation", "bug"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		parsed := intakeSettingsFromGraph(graph, newSpec(expression))

		assert.Equal(t, []string{"documentation", "bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterInclude, parsed.LabelFilterMode)
	})

	t.Run("round-trips the exclude mode", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"bug"}
		settings.LabelFilterMode = intakeLabelFilterExclude
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		parsed := intakeSettingsFromGraph(graph, newSpec(expression))

		assert.Equal(t, []string{"bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterExclude, parsed.LabelFilterMode)
	})

	// Canvases built before the expression was valid expr-lang still hold the
	// old form. Read those labels back so opening the settings does not drop
	// them; saving rewrites the expression.
	t.Run("reads labels from the legacy expression", func(t *testing.T) {
		legacy := `root().data.issue.labels.exists(label, label.name in ["documentation","bug"])`

		parsed := intakeSettingsFromGraph(graph, newSpec(legacy))

		assert.Equal(t, []string{"documentation", "bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterInclude, parsed.LabelFilterMode)
	})

	t.Run("reads the exclude mode from the legacy expression", func(t *testing.T) {
		legacy := `!(root().data.issue.labels.exists(label, label.name in ["bug"]))`

		parsed := intakeSettingsFromGraph(graph, newSpec(legacy))

		assert.Equal(t, []string{"bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterExclude, parsed.LabelFilterMode)
	})

	t.Run("reads the assignment from the legacy expression", func(t *testing.T) {
		parsed := intakeSettingsFromGraph(graph, newSpec(intakeLegacyUnassignedCondition))
		assert.Equal(t, intakeAssignmentUnassigned, parsed.Assignment)

		parsed = intakeSettingsFromGraph(graph, newSpec(intakeLegacyAssignedCondition))
		assert.Equal(t, intakeAssignmentAssigned, parsed.Assignment)
	})

	t.Run("round-trips the assignment through build and parse", func(t *testing.T) {
		for _, assignment := range []string{intakeAssignmentAssigned, intakeAssignmentUnassigned} {
			settings := defaultIntakeSettings()
			settings.Assignment = assignment
			expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

			parsed := intakeSettingsFromGraph(graph, newSpec(expression))

			assert.Equal(t, assignment, parsed.Assignment)
		}
	})
}
