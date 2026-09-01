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
		}, canvas.Spec.Edges)

		trigger := findSpecNode(t, canvas, backlogTriggerNodeID)
		assert.Equal(t, factory.OnWorkOrderTriggerName, trigger.Component)

		report := findSpecNode(t, canvas, intakeReportConfidenceNodeID)
		assert.Equal(t, intakeReportConfidenceComponent, report.Component)
		assert.Equal(t, "{{ root().data.workOrder.id }}", report.Configuration["orderId"])
		assert.Equal(t, "confidence", report.Configuration["checkKey"])
		assert.Equal(t, "Confidence score", report.Configuration["name"])
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
