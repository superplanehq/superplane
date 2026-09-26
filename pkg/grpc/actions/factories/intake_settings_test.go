package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/proto"
)

func Test__intakeFilterExpressionFor_AuthorsWithAccess(t *testing.T) {
	t.Run("off by default", func(t *testing.T) {
		settings := defaultIntakeSettings()
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)
		assert.NotContains(t, expression, intakeAuthorAccessCondition)
	})

	t.Run("does not use webhook author association when on", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.AuthorsWithAccess = true

		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)
		assert.NotContains(t, expression, "author_association")
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

	t.Run("reads true from the repository permission node", func(t *testing.T) {
		graph := intakeGraph{
			FilterNodeID:           "filter",
			AuthorPermissionNodeID: intakeAuthorPermissionNodeID,
		}
		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec("true"))

		assert.True(t, parsed.AuthorsWithAccess)
	})

	t.Run("reads false when the condition is absent", func(t *testing.T) {
		settings := defaultIntakeSettings()
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		graph := intakeGraph{FilterNodeID: "filter"}
		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(expression))

		assert.False(t, parsed.AuthorsWithAccess)
	})

	t.Run("a hand-edited expression falls back to the default", func(t *testing.T) {
		graph := intakeGraph{FilterNodeID: "filter"}
		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(`root().data.issue.author_association == "OWNER"`))

		assert.False(t, parsed.AuthorsWithAccess)
	})

	t.Run("reads the legacy webhook condition", func(t *testing.T) {
		graph := intakeGraph{FilterNodeID: "filter"}
		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(intakeAuthorAccessCondition))

		assert.True(t, parsed.AuthorsWithAccess)
	})
}

func Test__intakeTriggerActionsFor(t *testing.T) {
	t.Run("listens for opened and reopened issues by default", func(t *testing.T) {
		settings := defaultIntakeSettings()

		assert.Equal(t, []any{"opened", "reopened", "labeled"}, intakeTriggerActionsFor(settings))
	})

	t.Run("listens only for opened issues", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.ReopenedIssues = false
		settings.SuperplaneLabelAdded = false

		assert.Equal(t, []any{"opened"}, intakeTriggerActionsFor(settings))
	})

	t.Run("listens only for reopened issues", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.NewIssues = false
		settings.SuperplaneLabelAdded = false

		assert.Equal(t, []any{"reopened"}, intakeTriggerActionsFor(settings))
	})

	t.Run("can turn the label trigger off", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.SuperplaneLabelAdded = false

		assert.Equal(t, []any{"opened", "reopened"}, intakeTriggerActionsFor(settings))
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
		settings := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec([]any{"labeled"}))
		assert.False(t, settings.NewIssues)
		assert.False(t, settings.ReopenedIssues)
		assert.True(t, settings.SuperplaneLabelAdded)

		settings = intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec([]any{"opened"}))
		assert.True(t, settings.NewIssues)
		assert.False(t, settings.ReopenedIssues)

		settings = intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec([]any{"reopened"}))
		assert.False(t, settings.NewIssues)
		assert.True(t, settings.ReopenedIssues)
	})

	t.Run("round-trips every combination through build and parse", func(t *testing.T) {
		for _, newIssues := range []bool{true, false} {
			for _, reopenedIssues := range []bool{true, false} {
				settings := defaultIntakeSettings()
				settings.NewIssues = newIssues
				settings.ReopenedIssues = reopenedIssues

				parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(intakeTriggerActionsFor(settings)))

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

		assert.True(t, intakeSettingsChangeTrigger(models.FactoryIntakeSourceGitHubIssues, current, updated))
	})

	t.Run("sees no change when the toggles match", func(t *testing.T) {
		current := defaultIntakeSettings()

		assert.False(t, intakeSettingsChangeTrigger(models.FactoryIntakeSourceGitHubIssues, current, current))
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

		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(expression))

		assert.Equal(t, []string{"documentation", "bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterInclude, parsed.LabelFilterMode)
	})

	t.Run("round-trips the exclude mode", func(t *testing.T) {
		settings := defaultIntakeSettings()
		settings.Labels = []string{"bug"}
		settings.LabelFilterMode = intakeLabelFilterExclude
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(expression))

		assert.Equal(t, []string{"bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterExclude, parsed.LabelFilterMode)
	})

	// Canvases built before the expression was valid expr-lang still hold the
	// old form. Read those labels back so opening the settings does not drop
	// them; saving rewrites the expression.
	t.Run("reads labels from the legacy expression", func(t *testing.T) {
		legacy := `root().data.issue.labels.exists(label, label.name in ["documentation","bug"])`

		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(legacy))

		assert.Equal(t, []string{"documentation", "bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterInclude, parsed.LabelFilterMode)
	})

	t.Run("reads the exclude mode from the legacy expression", func(t *testing.T) {
		legacy := `!(root().data.issue.labels.exists(label, label.name in ["bug"]))`

		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(legacy))

		assert.Equal(t, []string{"bug"}, parsed.Labels)
		assert.Equal(t, intakeLabelFilterExclude, parsed.LabelFilterMode)
	})

	t.Run("reads the assignment from the legacy expression", func(t *testing.T) {
		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(intakeLegacyUnassignedCondition))
		assert.Equal(t, intakeAssignmentUnassigned, parsed.Assignment)

		parsed = intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(intakeLegacyAssignedCondition))
		assert.Equal(t, intakeAssignmentAssigned, parsed.Assignment)
	})

	t.Run("round-trips the assignment through build and parse", func(t *testing.T) {
		for _, assignment := range []string{intakeAssignmentAssigned, intakeAssignmentUnassigned} {
			settings := defaultIntakeSettings()
			settings.Assignment = assignment
			expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, settings)

			parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceGitHubIssues, graph, newSpec(expression))

			assert.Equal(t, assignment, parsed.Assignment)
		}
	})
}

func Test__intakeSentryActionsFor(t *testing.T) {
	t.Run("listens for created issues by default", func(t *testing.T) {
		assert.Equal(t, []any{"created"}, intakeSentryActionsFor(defaultSentryIntakeSettings()))
	})

	t.Run("maps each event checkbox to its webhook action", func(t *testing.T) {
		settings := defaultSentryIntakeSettings()
		settings.SentryNewIssues = false
		settings.SentryRegressedIssues = false
		settings.SentryAssignedIssues = true

		assert.Equal(t, []any{"assigned"}, intakeSentryActionsFor(settings))
	})

	t.Run("an empty selection lists no webhook actions", func(t *testing.T) {
		settings := defaultSentryIntakeSettings()
		settings.SentryNewIssues = false
		settings.SentryRegressedIssues = false

		assert.Empty(t, intakeSentryActionsFor(settings))
	})
}

func Test__intakeFilterExpressionFor_SentryLevels(t *testing.T) {
	t.Run("accepts every level when none are selected", func(t *testing.T) {
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceSentryExceptions, defaultSentryIntakeSettings())
		assert.Equal(t, "true", expression)
	})

	t.Run("builds a level membership check in a stable order", func(t *testing.T) {
		settings := defaultSentryIntakeSettings()
		settings.SentryLevels = []string{"error", "fatal", "unknown"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceSentryExceptions, settings)

		assert.Equal(t, `(root().data.data.issue?.level ?? "") in ["fatal","error"]`, expression)
	})
}

func Test__intakeFilterExpressionFor_DependabotSeverities(t *testing.T) {
	t.Run("accepts every severity when none are selected", func(t *testing.T) {
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceDependabotAlerts, defaultDependabotIntakeSettings())
		assert.Equal(t, "true", expression)
	})

	t.Run("builds a severity membership check in a stable order", func(t *testing.T) {
		settings := defaultDependabotIntakeSettings()
		settings.DependabotSeverities = []string{"low", "critical", "unknown"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceDependabotAlerts, settings)

		assert.Equal(t, `(root().data.alert.security_advisory.severity ?? "") in ["critical","low"]`, expression)
	})

	t.Run("treats every known severity as all severities", func(t *testing.T) {
		settings := defaultDependabotIntakeSettings()
		settings.DependabotSeverities = []string{"low", "medium", "high", "critical"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceDependabotAlerts, settings)

		assert.Equal(t, "true", expression)
	})
}

func Test__intakeSettingsFromGraph_Sentry(t *testing.T) {
	newSpec := func(actions []any, expression string) models.LiveCanvasSpec {
		return models.LiveCanvasSpec{
			Nodes: []models.Node{
				{
					ID:            intakeTriggerNodeID,
					Configuration: map[string]any{"actions": actions},
				},
				{
					ID:            intakeFilterNodeID,
					Configuration: map[string]any{"expression": expression},
				},
			},
		}
	}
	graph := intakeGraph{TriggerNodeID: intakeTriggerNodeID, FilterNodeID: intakeFilterNodeID}

	t.Run("reads the trigger actions and level list", func(t *testing.T) {
		settings := defaultSentryIntakeSettings()
		settings.SentryRegressedIssues = true
		settings.SentryAssignedIssues = true
		settings.SentryLevels = []string{"warning", "error"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceSentryExceptions, settings)

		parsed := intakeSettingsFromGraph(
			models.FactoryIntakeSourceSentryExceptions,
			graph,
			newSpec(intakeSentryActionsFor(settings), expression),
		)

		assert.True(t, parsed.SentryNewIssues)
		assert.True(t, parsed.SentryRegressedIssues)
		assert.True(t, parsed.SentryAssignedIssues)
		assert.Equal(t, []string{"error", "warning"}, parsed.SentryLevels)
	})

	t.Run("a hand-edited expression falls back to the defaults", func(t *testing.T) {
		parsed := intakeSettingsFromGraph(
			models.FactoryIntakeSourceSentryExceptions,
			graph,
			newSpec([]any{"created", "unresolved"}, `root().data.data.issue.level == "error"`),
		)

		assert.True(t, parsed.SentryNewIssues)
		assert.True(t, parsed.SentryRegressedIssues)
		assert.False(t, parsed.SentryAssignedIssues)
		assert.Empty(t, parsed.SentryLevels)
	})

	t.Run("an intake without a filter reports the default settings", func(t *testing.T) {
		parsed := intakeSettingsFromGraph(
			models.FactoryIntakeSourceSentryExceptions,
			intakeGraph{TriggerNodeID: intakeTriggerNodeID},
			models.LiveCanvasSpec{
				Nodes: []models.Node{
					{
						ID:            intakeTriggerNodeID,
						Configuration: map[string]any{"actions": []any{"created"}},
					},
				},
			},
		)

		assert.Equal(t, defaultSentryIntakeSettings().SentryNewIssues, parsed.SentryNewIssues)
		assert.Equal(t, defaultSentryIntakeSettings().SentryRegressedIssues, parsed.SentryRegressedIssues)
		assert.False(t, parsed.SentryAssignedIssues)
		assert.Empty(t, parsed.SentryLevels)
	})
}

func Test__intakeFilterExpressionFor_EvaluatesAgainstSentryPayloads(t *testing.T) {
	errorIssue := map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"title": "Error #1: This is a test error!",
				"level": "error",
			},
		},
	}
	warningIssue := map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"title": "Slow query",
				"level": "warning",
			},
		},
	}
	issueWithoutLevel := map[string]any{
		"data": map[string]any{
			"issue": map[string]any{
				"title": "Error #1: This is a test error!",
			},
		},
	}

	t.Run("accepts every level when none are selected", func(t *testing.T) {
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceSentryExceptions, defaultSentryIntakeSettings())

		assert.Equal(t, true, evalRootDataExpression(t, expression, errorIssue))
		assert.Equal(t, true, evalRootDataExpression(t, expression, warningIssue))
		assert.Equal(t, true, evalRootDataExpression(t, expression, issueWithoutLevel))
	})

	t.Run("keeps only the selected levels and treats a missing level as no match", func(t *testing.T) {
		settings := defaultSentryIntakeSettings()
		settings.SentryLevels = []string{"fatal", "error"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceSentryExceptions, settings)

		assert.Equal(t, true, evalRootDataExpression(t, expression, errorIssue))
		assert.Equal(t, false, evalRootDataExpression(t, expression, warningIssue))
		assert.Equal(t, false, evalRootDataExpression(t, expression, issueWithoutLevel))
	})
}

func Test__applyIntakeSettingsToGraph_Sentry(t *testing.T) {
	legacyNodes := func() []models.Node {
		return []models.Node{
			{
				ID:            intakeTriggerNodeID,
				Configuration: map[string]any{"actions": []any{"created", "unresolved"}},
			},
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
	}
	legacyEdges := []models.Edge{
		{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID},
	}
	legacyGraph := intakeGraph{TriggerNodeID: intakeTriggerNodeID, CreateNodeID: intakeCreateNodeID}

	t.Run("inserts a filter when a legacy intake selects a level", func(t *testing.T) {
		nodes, edges, err := applyIntakeSettingsToGraph(
			models.FactoryIntakeSourceSentryExceptions,
			legacyGraph,
			models.LiveCanvasSpec{Nodes: legacyNodes(), Edges: legacyEdges},
			&pb.FactoryIntake_Settings{SentryLevels: []string{"error"}},
			legacyNodes(),
			append([]models.Edge{}, legacyEdges...),
		)
		require.NoError(t, err)

		filter := findModelNode(t, nodes, intakeFilterNodeID)
		assert.Equal(t, `(root().data.data.issue?.level ?? "") in ["error"]`, filter.Configuration["expression"])
		assert.ElementsMatch(t, []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, edges)
	})

	t.Run("saves an empty event list instead of listening for every action", func(t *testing.T) {
		nodes := []models.Node{
			{
				ID:            intakeTriggerNodeID,
				Configuration: map[string]any{"actions": []any{"created", "unresolved"}},
			},
			{
				ID:            intakeFilterNodeID,
				Configuration: map[string]any{"expression": "true"},
			},
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
		graph := intakeGraph{
			TriggerNodeID: intakeTriggerNodeID,
			FilterNodeID:  intakeFilterNodeID,
			CreateNodeID:  intakeCreateNodeID,
		}

		updated, _, err := applyIntakeSettingsToGraph(
			models.FactoryIntakeSourceSentryExceptions,
			graph,
			models.LiveCanvasSpec{Nodes: nodes},
			&pb.FactoryIntake_Settings{
				SentryNewIssues:       proto.Bool(false),
				SentryRegressedIssues: proto.Bool(false),
				SentryAssignedIssues:  proto.Bool(false),
			},
			nodes,
			nil,
		)
		require.NoError(t, err)

		trigger := findModelNode(t, updated, intakeTriggerNodeID)
		assert.Empty(t, trigger.Configuration["actions"])
	})
}

func Test__intakeInstructions(t *testing.T) {
	createNode := func(instructions string) models.Node {
		node := componentNode(intakeCreateNodeID, intakeCreateComponent)
		node.Configuration = map[string]any{"title": "{{ root().data.issue.title }}"}
		if instructions != "" {
			node.Configuration[intakeInstructionsConfigurationKey] = instructions
		}
		return node
	}
	graph := intakeGraph{TriggerNodeID: intakeTriggerNodeID, CreateNodeID: intakeCreateNodeID}

	t.Run("a new Dependabot intake carries the default instructions", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceDependabotAlerts})
		require.NoError(t, err)

		create := findSpecNode(t, canvas, intakeCreateNodeID)
		assert.Equal(t, dependabotDefaultInstructions, create.Configuration[intakeInstructionsConfigurationKey])
		assert.Contains(t, dependabotDefaultInstructions, "update that direct dependency")
	})

	t.Run("other sources start without instructions", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues})
		require.NoError(t, err)

		create := findSpecNode(t, canvas, intakeCreateNodeID)
		assert.NotContains(t, create.Configuration, intakeInstructionsConfigurationKey)
	})

	t.Run("the request wins over the source default", func(t *testing.T) {
		settings := parseIntakeSettings(defaultDependabotIntakeSettings(), &pb.FactoryIntake_Settings{
			Instructions: proto.String("  Open one pull request per manifest.  "),
		})
		assert.Equal(t, "Open one pull request per manifest.", settings.Instructions)

		kept := parseIntakeSettings(defaultDependabotIntakeSettings(), &pb.FactoryIntake_Settings{})
		assert.Equal(t, dependabotDefaultInstructions, kept.Instructions)

		cleared := parseIntakeSettings(defaultDependabotIntakeSettings(), &pb.FactoryIntake_Settings{Instructions: proto.String("")})
		assert.Empty(t, cleared.Instructions)
	})

	t.Run("reads the instructions off the Create Task node, not the default", func(t *testing.T) {
		spec := models.LiveCanvasSpec{Nodes: []models.Node{
			triggerNode(intakeTriggerNodeID, "github.onDependabotAlert"),
			createNode("Ask before a major version update."),
		}}
		settings := intakeSettingsFromGraph(models.FactoryIntakeSourceDependabotAlerts, graph, spec)
		assert.Equal(t, "Ask before a major version update.", settings.Instructions)

		legacy := models.LiveCanvasSpec{Nodes: []models.Node{
			triggerNode(intakeTriggerNodeID, "github.onDependabotAlert"),
			createNode(""),
		}}
		assert.Empty(t, intakeSettingsFromGraph(models.FactoryIntakeSourceDependabotAlerts, graph, legacy).Instructions)

		serialized := serializeIntakeSettings(models.FactoryIntakeSourceDependabotAlerts, settings)
		assert.Equal(t, "Ask before a major version update.", serialized.GetInstructions())
	})

	t.Run("an update writes the instructions on the Create Task node", func(t *testing.T) {
		nodes := []models.Node{
			triggerNode(intakeTriggerNodeID, "github.onIssue"),
			componentNode(intakeFilterNodeID, intakeFilterComponent),
			createNode(""),
		}
		fullGraph := intakeGraph{TriggerNodeID: intakeTriggerNodeID, FilterNodeID: intakeFilterNodeID, CreateNodeID: intakeCreateNodeID}

		updated, _, err := applyIntakeSettingsToGraph(
			models.FactoryIntakeSourceGitHubIssues,
			fullGraph,
			models.LiveCanvasSpec{Nodes: nodes},
			&pb.FactoryIntake_Settings{Instructions: proto.String("Keep the change small.")},
			nodes,
			nil,
		)
		require.NoError(t, err)
		create := findModelNode(t, updated, intakeCreateNodeID)
		assert.Equal(t, "Keep the change small.", create.Configuration[intakeInstructionsConfigurationKey])
		assert.Equal(t, "{{ root().data.issue.title }}", create.Configuration["title"])

		cleared, _, err := applyIntakeSettingsToGraph(
			models.FactoryIntakeSourceGitHubIssues,
			fullGraph,
			models.LiveCanvasSpec{Nodes: updated},
			&pb.FactoryIntake_Settings{Instructions: proto.String("")},
			updated,
			nil,
		)
		require.NoError(t, err)
		assert.NotContains(t, findModelNode(t, cleared, intakeCreateNodeID).Configuration, intakeInstructionsConfigurationKey)
	})
}

func Test__intakeFilterExpressionFor_ProductiveKeyTasks(t *testing.T) {
	t.Run("excludes key tasks by default", func(t *testing.T) {
		expression := intakeFilterExpressionFor(
			models.FactoryIntakeSourceProductiveTasks,
			defaultProductiveIntakeSettings(),
		)
		assert.Equal(t, intakeProductiveExcludeKeyTasksCondition, expression)
	})

	t.Run("passes every task when the filter is off", func(t *testing.T) {
		settings := defaultProductiveIntakeSettings()
		settings.ExcludeKeyTasks = false
		assert.Equal(t, "true", intakeFilterExpressionFor(models.FactoryIntakeSourceProductiveTasks, settings))
	})

	regularTask := map[string]any{
		"data": map[string]any{
			"attributes": map[string]any{"type_id": 1, "title": "Fix payment retries"},
		},
	}
	keyTask := map[string]any{
		"data": map[string]any{
			"attributes": map[string]any{"type_id": 3, "title": "Key task 1"},
		},
	}

	t.Run("drops a key task and keeps a regular task", func(t *testing.T) {
		expression := intakeFilterExpressionFor(
			models.FactoryIntakeSourceProductiveTasks,
			defaultProductiveIntakeSettings(),
		)
		assert.Equal(t, true, evalRootDataExpression(t, expression, regularTask))
		assert.Equal(t, false, evalRootDataExpression(t, expression, keyTask))
	})

	t.Run("keeps a task only when it is in a selected task list", func(t *testing.T) {
		settings := defaultProductiveIntakeSettings()
		settings.ExcludeKeyTasks = false
		settings.TaskListIDs = []string{"list-a", "list-b"}
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceProductiveTasks, settings)
		assert.Equal(
			t,
			`(root().data.data.relationships.task_list.data.id ?? "") in ["list-a","list-b"]`,
			expression,
		)

		inList := map[string]any{
			"data": map[string]any{
				"relationships": map[string]any{
					"task_list": map[string]any{"data": map[string]any{"id": "list-a"}},
				},
			},
		}
		otherList := map[string]any{
			"data": map[string]any{
				"relationships": map[string]any{
					"task_list": map[string]any{"data": map[string]any{"id": "list-c"}},
				},
			},
		}
		assert.Equal(t, true, evalRootDataExpression(t, expression, inList))
		assert.Equal(t, false, evalRootDataExpression(t, expression, otherList))

		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceProductiveTasks, intakeGraph{FilterNodeID: intakeFilterNodeID}, models.LiveCanvasSpec{
			Nodes: []models.Node{{
				ID:            intakeFilterNodeID,
				Configuration: map[string]any{"expression": expression},
			}},
		})
		assert.Equal(t, []string{"list-a", "list-b"}, parsed.TaskListIDs)
	})
}

func Test__intakeSettingsFromGraph_ProductiveKeyTasks(t *testing.T) {
	graph := intakeGraph{FilterNodeID: intakeFilterNodeID}

	t.Run("defaults to excluding key tasks when the filter is absent", func(t *testing.T) {
		parsed := intakeSettingsFromGraph(models.FactoryIntakeSourceProductiveTasks, intakeGraph{}, models.LiveCanvasSpec{})
		assert.True(t, parsed.ExcludeKeyTasks)
	})

	t.Run("reads the filter expression", func(t *testing.T) {
		on := intakeSettingsFromGraph(models.FactoryIntakeSourceProductiveTasks, graph, models.LiveCanvasSpec{
			Nodes: []models.Node{{
				ID:            intakeFilterNodeID,
				Configuration: map[string]any{"expression": intakeProductiveExcludeKeyTasksCondition},
			}},
		})
		assert.True(t, on.ExcludeKeyTasks)

		off := intakeSettingsFromGraph(models.FactoryIntakeSourceProductiveTasks, graph, models.LiveCanvasSpec{
			Nodes: []models.Node{{
				ID:            intakeFilterNodeID,
				Configuration: map[string]any{"expression": "true"},
			}},
		})
		assert.False(t, off.ExcludeKeyTasks)
	})
}

func Test__serializeIntakeSettings_ProductiveKeyTasks(t *testing.T) {
	settings := defaultProductiveIntakeSettings()
	serialized := serializeIntakeSettings(models.FactoryIntakeSourceProductiveTasks, settings)
	assert.True(t, serialized.GetExcludeKeyTasks())

	settings.ExcludeKeyTasks = false
	serialized = serializeIntakeSettings(models.FactoryIntakeSourceProductiveTasks, settings)
	assert.False(t, serialized.GetExcludeKeyTasks())
}

func Test__applyIntakeSettingsToGraph_Productive(t *testing.T) {
	legacyNodes := func() []models.Node {
		return []models.Node{
			{
				ID:            intakeTriggerNodeID,
				Configuration: map[string]any{"actions": []any{"created"}},
			},
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
	}
	legacyEdges := []models.Edge{
		{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID},
	}
	legacyGraph := intakeGraph{TriggerNodeID: intakeTriggerNodeID, CreateNodeID: intakeCreateNodeID}

	t.Run("inserts a filter that drops key tasks", func(t *testing.T) {
		nodes, edges, err := applyIntakeSettingsToGraph(
			models.FactoryIntakeSourceProductiveTasks,
			legacyGraph,
			models.LiveCanvasSpec{Nodes: legacyNodes(), Edges: legacyEdges},
			&pb.FactoryIntake_Settings{ExcludeKeyTasks: proto.Bool(true)},
			legacyNodes(),
			append([]models.Edge{}, legacyEdges...),
		)
		require.NoError(t, err)

		filter := findModelNode(t, nodes, intakeFilterNodeID)
		assert.Equal(t, intakeProductiveExcludeKeyTasksCondition, filter.Configuration["expression"])
		assert.ElementsMatch(t, []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, edges)
	})
}
