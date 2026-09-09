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
	t.Run("listens for opened and reopened issues by default", func(t *testing.T) {
		settings := defaultIntakeSettings()

		assert.Equal(t, []any{"opened", "reopened"}, intakeTriggerActionsFor(settings))
	})

	t.Run("listens only for opened issues", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.ReopenedIssues = false

		assert.Equal(t, []any{"opened"}, intakeTriggerActionsFor(settings))
	})

	t.Run("listens only for reopened issues", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.NewIssues = false

		assert.Equal(t, []any{"reopened"}, intakeTriggerActionsFor(settings))
	})

	t.Run("also listens for added labels", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.SuperplaneLabelAdded = true

		assert.Equal(t, []any{"opened", "reopened", "labeled"}, intakeTriggerActionsFor(settings))
	})

	t.Run("can listen only for added labels", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.NewIssues = false
		settings.ReopenedIssues = false
		settings.SuperplaneLabelAdded = true

		assert.Equal(t, []any{"labeled"}, intakeTriggerActionsFor(settings))
	})
}

func Test__intakeSettingsFromGraph_TriggerActions(t *testing.T) {
	newSpec := func(actions []any) models.LiveCanvasSpec {
		return models.LiveCanvasSpec{
			Nodes: []models.Node{
				{
					ID:            intakeTriggerNodeID,
					Configuration: map[string]any{"actions": actions},
				},
			},
		}
	}
	graph := intakeGraph{TriggerNodeID: intakeTriggerNodeID}

	t.Run("reads each action on its own", func(t *testing.T) {
		settings := intakeSettingsFromGraph(graph, newSpec([]any{"labeled"}))
		assert.False(t, settings.NewIssues)
		assert.False(t, settings.ReopenedIssues)
		assert.True(t, settings.SuperplaneLabelAdded)

		settings = intakeSettingsFromGraph(graph, newSpec([]any{"opened"}))
		assert.True(t, settings.NewIssues)
		assert.False(t, settings.ReopenedIssues)

		settings = intakeSettingsFromGraph(graph, newSpec([]any{"reopened"}))
		assert.False(t, settings.NewIssues)
		assert.True(t, settings.ReopenedIssues)
	})

	t.Run("round-trips every combination through build and parse", func(t *testing.T) {
		for _, newIssues := range []bool{true, false} {
			for _, reopenedIssues := range []bool{true, false} {
				settings := defaultIntakeSettings()
				settings.NewIssues = newIssues
				settings.ReopenedIssues = reopenedIssues

				parsed := intakeSettingsFromGraph(graph, newSpec(intakeTriggerActionsFor(settings)))

				assert.Equal(t, newIssues, parsed.NewIssues)
				assert.Equal(t, reopenedIssues, parsed.ReopenedIssues)
			}
		}
	})
}

func Test__intakeSettingsChangeTrigger(t *testing.T) {
	t.Run("sees a change to the reopened toggle", func(t *testing.T) {
		current := defaultIntakeSettings()
		updated := current
		updated.ReopenedIssues = false

		assert.True(t, intakeSettingsChangeTrigger(current, updated))
	})

	t.Run("sees no change when the toggles match", func(t *testing.T) {
		current := defaultIntakeSettings()

		assert.False(t, intakeSettingsChangeTrigger(current, current))
	})
}

func Test__intakeFilterExpressionFor_SuperplaneLabelAdded(t *testing.T) {
	settings := defaultIntakeSettings()
	settings.SuperplaneLabelAdded = true

	expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

	assert.Contains(t, expression, intakeSuperplaneLabelCondition)
}

// The filter node runs the expression through expr-lang, so building it is not
// enough: it has to evaluate against the payloads GitHub actually sends. An
// `opened` payload carries no `label`, which is why every condition that
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
	superplaneLabelAdded := map[string]any{
		"action": "labeled",
		"label":  map[string]any{"name": "superplane"},
		"issue": map[string]any{
			"state":              "open",
			"author_association": "MEMBER",
			"assignees":          []any{},
			"labels": []any{
				map[string]any{"name": "documentation"},
				map[string]any{"name": "superplane"},
			},
		},
	}
	otherLabelAdded := map[string]any{
		"action": "labeled",
		"label":  map[string]any{"name": "chore"},
		"issue": map[string]any{
			"state":              "open",
			"author_association": "MEMBER",
			"assignees":          []any{},
			"labels": []any{
				map[string]any{"name": "documentation"},
				map[string]any{"name": "chore"},
			},
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

	t.Run("matches only the superplane label on a labeled event", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.SuperplaneLabelAdded = true
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, superplaneLabelAdded))
		assert.Equal(t, false, evalRootDataExpression(t, expression, otherLabelAdded))
		// An opened payload has no label, so the label check must not run.
		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithLabel))
	})

	t.Run("keeps the label filter for labeled events", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation"}
		settings.SuperplaneLabelAdded = true
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, superplaneLabelAdded))
		assert.Equal(t, false, evalRootDataExpression(t, expression, otherLabelAdded))
		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithLabel))
		assert.Equal(t, false, evalRootDataExpression(t, expression, openedWithoutLabel))
	})

	t.Run("combines every filter", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"documentation"}
		settings.Assignment = intakeAssignmentUnassigned
		settings.AuthorsWithAccess = true
		settings.SuperplaneLabelAdded = true
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, openedWithLabel))
		assert.Equal(t, true, evalRootDataExpression(t, expression, superplaneLabelAdded))
		assert.Equal(t, false, evalRootDataExpression(t, expression, otherLabelAdded))
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
