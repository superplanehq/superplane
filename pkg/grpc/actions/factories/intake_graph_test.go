package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
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

func Test__BuildBacklogCanvas(t *testing.T) {
	t.Run("the item flows from a new work order to the confidence check", func(t *testing.T) {
		canvas := buildBacklogCanvas(backlogCanvasRequest{})

		assert.Equal(t, backlogDefaultName, canvas.Metadata.Name)
		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: backlogTriggerNodeID, TargetID: intakeAnalysisNodeID},
			{Channel: "passed", SourceID: intakeAnalysisNodeID, TargetID: intakeReportConfidenceNodeID},
			{Channel: "passed", SourceID: intakeAnalysisNodeID, TargetID: "attach-intent"},
			{Channel: "failed", SourceID: intakeAnalysisNodeID, TargetID: intakeAddRunErrorNodeID},
		}, canvas.Spec.Edges)

		trigger := findSpecNode(t, canvas, backlogTriggerNodeID)
		assert.Equal(t, factory.OnWorkOrderTriggerName, trigger.Component)

		report := findSpecNode(t, canvas, intakeReportConfidenceNodeID)
		assert.Equal(t, intakeReportConfidenceComponent, report.Component)
		assert.Equal(t, "{{ root().data.workOrder.id }}", report.Configuration["orderId"])
		assert.Equal(t, "confidence", report.Configuration["checkKey"])
		assert.Equal(t, "Confidence score", report.Configuration["name"])

		intent := findSpecNode(t, canvas, "attach-intent")
		assert.Equal(t, factory.AddWorkOrderArtifactComponentName, intent.Component)
		assert.Equal(t, "{{ root().data.workOrder.id }}", intent.Configuration["orderId"])
		assert.Equal(t, "markdown", intent.Configuration["artifactType"])
		assert.Equal(t, "spec.md", intent.Configuration["title"])
		assert.Equal(t, `{{ $["Analyze intake"].data.result.intent }}`, intent.Configuration["body"])

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

		analysis := findSpecNode(t, canvas, intakeAnalysisNodeID)
		assert.Equal(t, "runnerCodex", analysis.Component)
		assert.Equal(t, map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": "acme-openai"},
		}, analysis.Configuration["credentials"])
		assert.Equal(t, "gpt-5", analysis.Configuration["model"])
		assert.Equal(t, runner.MachineTypeE1LargeAMD64, analysis.Configuration["machineType"])
		assert.Equal(t, []any{
			map[string]any{
				"source":      "integration",
				"integration": map[string]any{"name": "github"},
			},
		}, analysis.Configuration["environmentFrom"])
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
		}, analysis.Configuration["environment"])

		steps, ok := analysis.Configuration["steps"].([]any)
		require.True(t, ok)
		require.Len(t, steps, 3)
		clone, ok := steps[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Clone repository", clone["name"])
		assert.Contains(t, clone["command"], "rm -rf repo")
		assert.Contains(t, clone["command"], `git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo`)
		prompt, ok := steps[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "repo", prompt["workingDirectory"])
		assert.NotContains(t, prompt["prompt"], "/tmp/spec.md")
		assert.NotContains(t, prompt["prompt"], "propose_spec")
		assert.NotContains(t, prompt["prompt"], "propose_confidence")
		assert.Contains(t, prompt["prompt"], "Score from 0 through 5")
		assert.Contains(t, prompt["prompt"], "Write one sentence that explains the score")
		assert.Contains(t, prompt["prompt"], "ask the person a multiple-choice question")
		assert.NotContains(t, prompt["prompt"], "survey")
		assert.Contains(t, prompt["prompt"], "how suitable the work is for an agent")
		assert.NotContains(t, prompt["prompt"], "check copy")
		assert.Contains(t, prompt["prompt"], "## Executive summary")
		assert.Contains(t, prompt["prompt"], "## Problem")
		assert.Contains(t, prompt["prompt"], "## Scope")
		assert.Contains(t, prompt["prompt"], "## Outcome")
		assert.Contains(t, prompt["prompt"], "## Approach")
		assert.Contains(t, prompt["prompt"], "## Files and seams")
		assert.Contains(t, prompt["prompt"], "## Acceptance")
		assert.Contains(t, prompt["prompt"], "## Risks")
		assert.Contains(t, prompt["prompt"], "Copy this markdown shape. Do not add extra paragraphs")
		assert.Contains(t, prompt["prompt"], "### Need")
		assert.Contains(t, prompt["prompt"], "### Result")
		assert.Contains(t, prompt["prompt"], "### Stay the same")
		assert.Contains(t, prompt["prompt"], "Do not explain what the app is, what stack it uses, or how the repository is organized.")
		assert.Contains(t, prompt["prompt"], "write 80 to 150 lines after the title")
		assert.Contains(t, prompt["prompt"], "Write a specification, not a chat note.")
		assert.Contains(t, prompt["prompt"], "## Why not start")
		assert.Contains(t, prompt["prompt"], "## What would make this clear")
		assert.Contains(t, prompt["prompt"], "If the score is 2 through 5")
		assert.Contains(t, prompt["prompt"], "If the score is 2 or 3")
		assert.Contains(t, prompt["prompt"], "If the score is 0 or 1")
		assert.Contains(t, prompt["prompt"], "Do not add an Open questions section.")
		assert.NotContains(t, prompt["prompt"], "Proposed outcome")
		assert.NotContains(t, prompt["prompt"], "## How I understand this")
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

func componentNode(nodeID, component string) models.Node {
	return models.Node{
		ID:   nodeID,
		Name: nodeID,
		Type: models.NodeTypeComponent,
		Ref:  models.NodeRef{Component: &models.ComponentRef{Name: component}},
	}
}
