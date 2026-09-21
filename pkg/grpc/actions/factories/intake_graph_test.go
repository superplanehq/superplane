package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func Test__ResolveIntakeGraph(t *testing.T) {
	t.Run("resolves a generated GitHub graph", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues)

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Equal(t, intakeTriggerNodeID, graph.TriggerNodeID)
		assert.Equal(t, intakeFilterNodeID, graph.FilterNodeID)
		assert.Equal(t, intakeCreateNodeID, graph.CreateNodeID)
		assert.Empty(t, graph.AnalysisNodeID)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("a graph without a score step is healthy", func(t *testing.T) {
		spec := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "github.onIssue"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Empty(t, graph.AnalysisNodeID)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("resolves renamed nodes by component", func(t *testing.T) {
		spec := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode("listen-here", "github.onIssue"),
				componentNode("gate", intakeFilterComponent),
				componentNode("file-it", intakeCreateComponent),
			},
			Edges: []models.Edge{
				{SourceID: "listen-here", TargetID: "gate"},
				{SourceID: "gate", TargetID: "file-it"},
			},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Equal(t, "listen-here", graph.TriggerNodeID)
		assert.Equal(t, "file-it", graph.CreateNodeID)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("a disconnected graph is not healthy", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues)
		spec.Edges = nil

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.False(t, graph.Healthy(spec.Edges))
	})

	t.Run("extra nodes on the path keep the graph healthy", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues)
		spec.Nodes = append(spec.Nodes, componentNode("label-filter", "filter"))
		spec.Edges = []models.Edge{
			{SourceID: intakeTriggerNodeID, TargetID: "label-filter"},
			{SourceID: "label-filter", TargetID: intakeFilterNodeID},
			{SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("a legacy analysis graph stays healthy", func(t *testing.T) {
		spec := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "github.onIssue"),
				componentNode(intakeAnalysisNodeID, "runnerCodex"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{
				{SourceID: intakeTriggerNodeID, TargetID: intakeAnalysisNodeID},
				{SourceID: intakeAnalysisNodeID, TargetID: intakeCreateNodeID},
			},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Equal(t, intakeAnalysisNodeID, graph.AnalysisNodeID)
		assert.True(t, graph.Healthy(spec.Edges))
	})
}

func Test__jiraWebhookHasRemoteID(t *testing.T) {
	assert.False(t, jiraWebhookHasRemoteID(nil))
	assert.False(t, jiraWebhookHasRemoteID("ready"))
	assert.False(t, jiraWebhookHasRemoteID(map[string]any{}))
	assert.False(t, jiraWebhookHasRemoteID(map[string]any{"webhookId": nil}))
	assert.True(t, jiraWebhookHasRemoteID(map[string]any{"webhookId": float64(1000)}))
	assert.True(t, jiraWebhookHasRemoteID(map[string]any{"webhookId": int64(1000)}))
}

func Test__jiraIntakeWebhookReady(t *testing.T) {
	assert.False(t, jiraIntakeWebhookReady(nil, uuid.Nil, intakeTriggerNodeID))
	assert.False(t, jiraIntakeWebhookReady(nil, uuid.New(), ""))
}

func Test__IntakeTriggerBinding(t *testing.T) {
	integrationID := "int-ready"
	spec := models.LiveCanvasSpec{
		Nodes: []models.Node{
			triggerNodeWithIntegration(intakeTriggerNodeID, "sentry.onIssue", integrationID, "payments"),
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		},
		Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
	}

	graph := resolveIntakeGraph(models.FactoryIntakeSourceSentryExceptions, spec)
	assert.Equal(t, integrationID, graph.TriggerIntegrationID(spec))
	assert.Equal(t, "payments", graph.TriggerResourceID(spec))
}

func Test__IntakeHealth(t *testing.T) {
	intake := &models.FactoryIntake{Source: models.FactoryIntakeSourceSentryExceptions}
	integrationID := "int-ready"
	spec := models.LiveCanvasSpec{
		Nodes: []models.Node{
			triggerNodeWithIntegration(intakeTriggerNodeID, "sentry.onIssue", integrationID, "payments"),
			componentNode(intakeCreateNodeID, intakeCreateComponent),
		},
		Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
	}
	graph := resolveIntakeGraph(models.FactoryIntakeSourceSentryExceptions, spec)

	t.Run("a ready integration is healthy", func(t *testing.T) {
		assert.Equal(t, pb.FactoryIntake_HEALTH_OK, intakeHealth(nil, intake, graph, spec, map[string]string{
			integrationID: models.IntegrationStateReady,
		}))
	})

	t.Run("a missing integration needs repair", func(t *testing.T) {
		assert.Equal(
			t,
			pb.FactoryIntake_HEALTH_MISSING_INTEGRATION,
			intakeHealth(nil, intake, graph, spec, map[string]string{}),
		)
	})

	t.Run("a not-ready integration needs repair", func(t *testing.T) {
		assert.Equal(t, pb.FactoryIntake_HEALTH_INTEGRATION_NOT_READY, intakeHealth(nil, intake, graph, spec, map[string]string{
			integrationID: models.IntegrationStateError,
		}))
	})

	t.Run("a disconnected graph is broken", func(t *testing.T) {
		broken := spec
		broken.Edges = nil
		assert.Equal(t, pb.FactoryIntake_HEALTH_GRAPH_BROKEN, intakeHealth(nil, intake, graph, broken, map[string]string{
			integrationID: models.IntegrationStateReady,
		}))
	})

	t.Run("an unbound GitHub intake stays healthy", func(t *testing.T) {
		unbound := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "github.onIssue"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
		}
		github := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, unbound)
		assert.Equal(
			t,
			pb.FactoryIntake_HEALTH_OK,
			intakeHealth(nil, &models.FactoryIntake{Source: models.FactoryIntakeSourceGitHubIssues}, github, unbound, nil),
		)
	})

	t.Run("an unbound Sentry intake needs a connection", func(t *testing.T) {
		unbound := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "sentry.onIssue"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
		}
		graph := resolveIntakeGraph(models.FactoryIntakeSourceSentryExceptions, unbound)
		assert.Equal(
			t,
			pb.FactoryIntake_HEALTH_MISSING_INTEGRATION,
			intakeHealth(nil, intake, graph, unbound, nil),
		)
	})

	t.Run("an unbound Productive.io intake needs a connection", func(t *testing.T) {
		unbound := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "productive.onTask"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
		}
		graph := resolveIntakeGraph(models.FactoryIntakeSourceProductiveTasks, unbound)
		assert.Equal(
			t,
			pb.FactoryIntake_HEALTH_MISSING_INTEGRATION,
			intakeHealth(nil, &models.FactoryIntake{Source: models.FactoryIntakeSourceProductiveTasks}, graph, unbound, nil),
		)
	})

	t.Run("an unbound Jira intake needs a connection", func(t *testing.T) {
		unbound := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "jira.onIssue"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
		}
		graph := resolveIntakeGraph(models.FactoryIntakeSourceJiraIssues, unbound)
		assert.Equal(
			t,
			pb.FactoryIntake_HEALTH_MISSING_INTEGRATION,
			intakeHealth(nil, &models.FactoryIntake{Source: models.FactoryIntakeSourceJiraIssues}, graph, unbound, nil),
		)
	})
}

func Test__BuildBacklogCanvas(t *testing.T) {
	t.Run("the item follows the refinement feature snapshot", func(t *testing.T) {
		canvas := buildBacklogCanvas(backlogCanvasRequest{})

		assert.Equal(t, backlogDefaultName, canvas.Metadata.Name)
		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: backlogTriggerNodeID, TargetID: backlogRefinementFilterNodeID},
			{Channel: "true", SourceID: backlogRefinementFilterNodeID, TargetID: backlogRefinementNodeID},
			{Channel: "failed", SourceID: backlogRefinementNodeID, TargetID: intakeAddRunErrorNodeID},
		}, canvas.Spec.Edges)
		assert.Nil(t, findSpecNodeOrNil(canvas, intakeAnalysisNodeID))

		trigger := findSpecNode(t, canvas, backlogTriggerNodeID)
		assert.Equal(t, factory.OnWorkOrderTriggerName, trigger.Component)
		assert.Equal(t, map[string]any{
			"id":      models.FactoryAppTemplateBacklogID,
			"version": backlogTemplateVersion,
		}, trigger.Metadata[factoryTemplateMetadataKey])

		filter := findSpecNode(t, canvas, backlogRefinementFilterNodeID)
		assert.Equal(t, intakeFilterComponent, filter.Component)
		assert.Equal(t, "{{ root().data.taskRefinementEnabled == true }}", filter.Configuration["expression"])

		refinement := findSpecNode(t, canvas, backlogRefinementNodeID)
		assert.Equal(t, "Refine Task", refinement.Name)
		steps, ok := refinement.Configuration["steps"].([]any)
		require.True(t, ok)
		require.Len(t, steps, 2)
		prompt, ok := steps[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Refine Task", prompt["name"])
		assert.Contains(t, prompt["prompt"], runner.PlanningSessionUserPromptMarkdown())
		assert.NotContains(t, prompt["prompt"], runner.PlanningSessionProtocolMarkdown())
		assert.Contains(t, prompt["prompt"], "{{ root().data.workOrder }}")

		runError := findSpecNode(t, canvas, intakeAddRunErrorNodeID)
		assert.Equal(t, intakeAddRunErrorComponent, runError.Component)
		assert.Equal(t, intakeAddRunErrorMessage, runError.Configuration["message"])
	})

	t.Run("the analysis runner authenticates with the workspace agent", func(t *testing.T) {
		canvas := buildBacklogCanvas(backlogCanvasRequest{
			Agent: &intakeAgent{
				Component: "runnerCodex",
				Credentials: map[string]any{
					"source":      runner.CredentialsSourceIntegration,
					"integration": map[string]any{"name": "acme-openai"},
				},
			},
		})

		refinement := findSpecNode(t, canvas, backlogRefinementNodeID)
		assert.Equal(t, "runnerCodex", refinement.Component)
		assert.Equal(t, map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": "acme-openai"},
		}, refinement.Configuration["credentials"])
		assert.Equal(t, "gpt-5", refinement.Configuration["model"])
		assert.Equal(t, runner.MachineTypeE1LargeAMD64, refinement.Configuration["machineType"])
		assert.Equal(t, []any{
			map[string]any{
				"source":      "integration",
				"integration": map[string]any{"name": "github"},
			},
		}, refinement.Configuration["environmentFrom"])
		assert.Equal(t, []any{
			map[string]any{
				"name":        "REPO_URL",
				"value":       "{{ root().data.workOrder.repository_url }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "BASE",
				"value":       "{{ root().data.workOrder.default_branch }}",
				"valueSource": "literal",
			},
		}, refinement.Configuration["environment"])

		steps, ok := refinement.Configuration["steps"].([]any)
		require.True(t, ok)
		require.Len(t, steps, 2)
		prompt, ok := steps[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Refine Task", prompt["name"])
		assert.Contains(t, prompt["prompt"], runner.PlanningSessionUserPromptMarkdown())
		assert.Contains(t, prompt["prompt"], "{{ root().data.workOrder }}")
	})
}

func Test__IsDefaultLegacyBacklog(t *testing.T) {
	t.Run("accepts the generated version 1 behavior with moved nodes", func(t *testing.T) {
		legacy := buildLegacyBacklogCanvas(backlogCanvasRequest{})
		nodes := legacy.Nodes()
		nodes[0].Position = models.Position{X: 900, Y: 700}

		assert.True(t, isDefaultLegacyBacklog(nodes, legacy.Edges()))
	})

	t.Run("rejects a customized prompt", func(t *testing.T) {
		legacy := buildLegacyBacklogCanvas(backlogCanvasRequest{})
		nodes := legacy.Nodes()
		analysis := findModelNode(t, nodes, intakeAnalysisNodeID)
		steps := analysis.Configuration["steps"].([]any)
		steps[1].(map[string]any)["prompt"] = "Use the team's custom scoring rules."

		assert.False(t, isDefaultLegacyBacklog(nodes, legacy.Edges()))
	})

	t.Run("rejects a customized component", func(t *testing.T) {
		legacy := buildLegacyBacklogCanvas(backlogCanvasRequest{})
		nodes := legacy.Nodes()
		for i := range nodes {
			if nodes[i].ID == intakeReportConfidenceNodeID {
				nodes[i].Ref.Component.Name = "customReporter"
			}
		}

		assert.False(t, isDefaultLegacyBacklog(nodes, legacy.Edges()))
	})

	t.Run("rejects customized edges", func(t *testing.T) {
		legacy := buildLegacyBacklogCanvas(backlogCanvasRequest{})
		edges := legacy.Edges()
		edges[0].TargetID = intakeReportConfidenceNodeID

		assert.False(t, isDefaultLegacyBacklog(legacy.Nodes(), edges))
	})

	t.Run("accepts the generated version 2 behavior", func(t *testing.T) {
		v2 := buildV2BacklogCanvas(backlogCanvasRequest{})

		assert.True(t, isDefaultLegacyBacklog(v2.Nodes(), v2.Edges()))
	})

	t.Run("rejects version 3", func(t *testing.T) {
		current := buildBacklogCanvas(backlogCanvasRequest{})

		assert.False(t, isDefaultLegacyBacklog(current.Nodes(), current.Edges()))
	})
}

func intakeSpecFromTemplate(t *testing.T, source string) models.LiveCanvasSpec {
	t.Helper()

	canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: source})
	require.NoError(t, err)

	return models.LiveCanvasSpec{Nodes: canvas.Nodes(), Edges: canvas.Edges()}
}

func triggerNode(nodeID, component string) models.Node {
	return models.Node{
		ID:   nodeID,
		Name: nodeID,
		Type: models.NodeTypeTrigger,
		Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: component}},
	}
}

func triggerNodeWithIntegration(nodeID, component, integrationID, resourceID string) models.Node {
	node := triggerNode(nodeID, component)
	node.IntegrationID = &integrationID
	node.Configuration = map[string]any{"project": resourceID}
	return node
}

func componentNode(nodeID, component string) models.Node {
	return models.Node{
		ID:   nodeID,
		Name: nodeID,
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: component}},
	}
}
