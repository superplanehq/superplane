package factories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dependabotcomp "github.com/superplanehq/superplane/pkg/integrations/github/components/dependabot"
	ghdependabot "github.com/superplanehq/superplane/pkg/integrations/github/dependabot"
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
			models.FactoryIntakeSourceJiraIssues:         "jira.onIssue",
			models.FactoryIntakeSourceDependabotAlerts:   "github.onDependabotAlert",
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
		assert.Equal(t, intakeSuperplaneLabelCondition, filter.Configuration["expression"])
	})

	t.Run("Jira issues flow from the trigger through the filter to the work order", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceJiraIssues})
		require.NoError(t, err)

		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, canvas.Spec.Edges)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Equal(t, []any{"created", "updated"}, trigger.Configuration["events"])
		assert.Equal(t, true, trigger.Metadata[intakeMetadataJiraMoveOnComplete])
		assert.Equal(t, "", trigger.Metadata[intakeMetadataJiraCompletionColumn])
	})

	t.Run("a Jira intake stores the chosen completion column on the trigger", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{
			Source: models.FactoryIntakeSourceJiraIssues,
			Settings: intakeSettings{
				ConfidencePct:        DefaultIntakeConfidencePct,
				JiraMoveOnComplete:   true,
				JiraCompletionColumn: "QA",
			},
		})
		require.NoError(t, err)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Equal(t, true, trigger.Metadata[intakeMetadataJiraMoveOnComplete])
		assert.Equal(t, "QA", trigger.Metadata[intakeMetadataJiraCompletionColumn])
	})

	t.Run("a Jira work order reads the plain text description, not the raw document", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceJiraIssues})
		require.NoError(t, err)

		// Jira holds a description in Atlassian Document Format, which reads
		// as a Go map once a template interpolates it.
		create := findSpecNode(t, canvas, intakeCreateNodeID)
		assert.Equal(t, "{{ root().data.description }}", create.Configuration["description"])
		assert.NotContains(t, create.Configuration["description"], "fields.description")
	})

	t.Run("a Sentry work order reads the formatted issue payload, not the permalink", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceSentryExceptions})
		require.NoError(t, err)

		create := findSpecNode(t, canvas, intakeCreateNodeID)
		assert.Equal(t, "{{ root().data.data.issue.title }}", create.Configuration["title"])
		assert.Equal(t, "{{ root().data.description }}", create.Configuration["description"])
		assert.NotContains(t, create.Configuration["description"], "permalink")
	})

	t.Run("a Sentry issue flows from the trigger through the filter to the work order", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceSentryExceptions})
		require.NoError(t, err)

		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, canvas.Spec.Edges)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Equal(t, intakeSentryActionsFor(defaultSentryIntakeSettings()), trigger.Configuration["actions"])

		filter := findSpecNode(t, canvas, intakeFilterNodeID)
		assert.Equal(t, intakeFilterComponent, filter.Component)
		assert.Equal(t, "true", filter.Configuration["expression"])
	})

	t.Run("a Dependabot work order matches the Go copy so later alerts merge in", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceDependabotAlerts})
		require.NoError(t, err)
		create := findSpecNode(t, canvas, intakeCreateNodeID)

		example := (&dependabotcomp.OnAlert{}).ExampleData()
		data, ok := example["data"].(map[string]any)
		require.True(t, ok)
		alert, ok := data["alert"].(map[string]any)
		require.True(t, ok)
		ref, ok := ghdependabot.PackageRefFromEventData(example)
		require.True(t, ok)

		title := evalRootDataExpression(t, templateExpressionSource(t, create.Configuration["title"].(string)), data)
		assert.Equal(t, ghdependabot.TaskTitle(ref), title)

		description := evalRootDataExpression(t, templateExpressionSource(t, create.Configuration["description"].(string)), data)
		require.IsType(t, "", description)
		assert.True(t, strings.HasSuffix(description.(string), "\n\n"+ghdependabot.AlertSection(alert)), description)
		assert.Contains(t, description, "Relationship: transitive")
	})

	t.Run("PagerDuty creates a work order without a filter", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourcePagerDutyIncidents})
		require.NoError(t, err)
		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID},
		}, canvas.Spec.Edges)
		assert.Nil(t, findSpecNodeOrNil(canvas, intakeFilterNodeID))
	})

	t.Run("Productive.io filters key tasks by default", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceProductiveTasks})
		require.NoError(t, err)
		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, canvas.Spec.Edges)
		filter := findSpecNode(t, canvas, intakeFilterNodeID)
		assert.Equal(t, intakeProductiveExcludeKeyTasksCondition, filter.Configuration["expression"])
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
		assert.Equal(t, []any{"opened", "reopened", "labeled"}, trigger.Configuration["actions"])
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
	t.Run("a default GitHub filter matches the superplane label event", func(t *testing.T) {
		assert.Equal(t, intakeSuperplaneLabelCondition, intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, defaultIntakeSettings()))
	})

	t.Run("GitHub labels and assignment join without a score", func(t *testing.T) {
		expression := intakeFilterExpressionFor(models.FactoryIntakeSourceGitHubIssues, intakeSettings{
			Labels:          []string{"bug"},
			LabelFilterMode: intakeLabelFilterExclude,
			Assignment:      intakeAssignmentUnassigned,
		})
		assert.NotContains(t, expression, ">=")
		assert.Contains(t, expression, `!(any(root().data.issue.labels, .name in ["bug"]))`)
		assert.Contains(t, expression, intakeUnassignedCondition)
	})
}

func Test__ensureIntakeFilterNode(t *testing.T) {
	t.Run("inserts a filter between the trigger and the work order", func(t *testing.T) {
		nodes := []models.Node{
			triggerNode(intakeTriggerNodeID, "sentry.onIssue"),
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
		edges := []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID},
		}
		graph := intakeGraph{TriggerNodeID: intakeTriggerNodeID, CreateNodeID: intakeCreateNodeID}

		nodes, edges, graph, err := ensureIntakeFilterNode(nodes, edges, graph)
		require.NoError(t, err)

		filter := findModelNode(t, nodes, intakeFilterNodeID)
		assert.Equal(t, intakeFilterComponent, filter.ComponentName())
		assert.Equal(t, "true", filter.Configuration["expression"])
		assert.Equal(t, intakeFilterNodeID, graph.FilterNodeID)
		assert.ElementsMatch(t, []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, edges)
	})

	t.Run("keeps an existing filter in place", func(t *testing.T) {
		nodes := []models.Node{
			triggerNode(intakeTriggerNodeID, "sentry.onIssue"),
			componentNode(intakeFilterNodeID, intakeFilterComponent),
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
		edges := []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}
		graph := intakeGraph{
			TriggerNodeID: intakeTriggerNodeID,
			FilterNodeID:  intakeFilterNodeID,
			CreateNodeID:  intakeCreateNodeID,
		}

		updatedNodes, updatedEdges, updatedGraph, err := ensureIntakeFilterNode(nodes, edges, graph)
		require.NoError(t, err)

		assert.Equal(t, nodes, updatedNodes)
		assert.Equal(t, edges, updatedEdges)
		assert.Equal(t, graph, updatedGraph)
	})

	t.Run("rejects a graph that cannot receive a filter", func(t *testing.T) {
		_, _, _, err := ensureIntakeFilterNode(nil, nil, intakeGraph{TriggerNodeID: intakeTriggerNodeID})
		require.EqualError(t, err, "intake automation has no filter to update")
	})
}

func Test__ConfigureIntakeAuthorAccess(t *testing.T) {
	t.Run("adds a repository permission gate", func(t *testing.T) {
		integrationID := "integration-1"
		nodes := []models.Node{
			{
				ID:            intakeTriggerNodeID,
				Name:          "On Issue",
				Type:          models.NodeTypeTrigger,
				Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}},
				Configuration: map[string]any{"repository": "acme/widgets"},
				IntegrationID: &integrationID,
			},
			componentNode(intakeFilterNodeID, intakeFilterComponent),
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
		edges := []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}

		nodes, edges, err := configureIntakeAuthorAccess(
			nodes,
			edges,
			resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, models.LiveCanvasSpec{Nodes: nodes, Edges: edges}),
			true,
		)
		require.NoError(t, err)

		permission := findModelNode(t, nodes, intakeAuthorPermissionNodeID)
		assert.Equal(t, intakeAuthorPermissionComponent, permission.ComponentName())
		assert.Equal(t, "acme/widgets", permission.Configuration["repository"])
		assert.Equal(t, "{{ root().data.issue.user.login }}", permission.Configuration["username"])
		assert.Equal(t, &integrationID, permission.IntegrationID)

		gate := findModelNode(t, nodes, intakeAuthorFilterNodeID)
		assert.Equal(t, intakeFilterComponent, gate.ComponentName())
		assert.Equal(t, `root().data.permission != "none"`, gate.Configuration["expression"])
		assert.ElementsMatch(t, []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeAuthorPermissionNodeID},
			{Channel: "default", SourceID: intakeAuthorPermissionNodeID, TargetID: intakeAuthorFilterNodeID},
			{Channel: "true", SourceID: intakeAuthorFilterNodeID, TargetID: intakeCreateNodeID},
		}, edges)
	})

	t.Run("removes the repository permission gate", func(t *testing.T) {
		nodes := []models.Node{
			triggerNode(intakeTriggerNodeID, "github.onIssue"),
			componentNode(intakeFilterNodeID, intakeFilterComponent),
			componentNode(intakeAuthorPermissionNodeID, intakeAuthorPermissionComponent),
			componentNode(intakeAuthorFilterNodeID, intakeFilterComponent),
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		}
		edges := []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeAuthorPermissionNodeID},
			{Channel: "default", SourceID: intakeAuthorPermissionNodeID, TargetID: intakeAuthorFilterNodeID},
			{Channel: "true", SourceID: intakeAuthorFilterNodeID, TargetID: intakeCreateNodeID},
		}

		nodes, edges, err := configureIntakeAuthorAccess(
			nodes,
			edges,
			resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, models.LiveCanvasSpec{Nodes: nodes, Edges: edges}),
			false,
		)
		require.NoError(t, err)

		assert.Nil(t, findModelNodeOrNil(nodes, intakeAuthorPermissionNodeID))
		assert.Nil(t, findModelNodeOrNil(nodes, intakeAuthorFilterNodeID))
		assert.ElementsMatch(t, []models.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}, edges)
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

func findModelNode(t *testing.T, nodes []models.Node, nodeID string) models.Node {
	t.Helper()

	node := findModelNodeOrNil(nodes, nodeID)
	require.NotNilf(t, node, "node %q not found", nodeID)
	return *node
}

func findModelNodeOrNil(nodes []models.Node, nodeID string) *models.Node {
	for i := range nodes {
		if nodes[i].ID == nodeID {
			return &nodes[i]
		}
	}
	return nil
}
