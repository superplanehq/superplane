package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__ResolveIntakeGraph(t *testing.T) {
	t.Run("resolves a generated graph and its threshold", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues, 80)

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Equal(t, intakeTriggerNodeID, graph.TriggerNodeID)
		assert.Equal(t, intakeAnalysisNodeID, graph.AnalysisNodeID)
		assert.Equal(t, intakeThresholdNodeID, graph.ThresholdNodeID)
		assert.Equal(t, intakeCreateNodeID, graph.CreateNodeID)
		assert.Equal(t, 80, graph.ConfidencePct)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("resolves renamed nodes by component", func(t *testing.T) {
		spec := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode("listen-here", "github.onIssue"),
				componentNode("score-it", "runnerCodex"),
				componentNode("gate", intakeThresholdComponent),
				componentNode("file-it", intakeCreateComponent),
			},
			Edges: []models.Edge{
				{SourceID: "listen-here", TargetID: "score-it"},
				{SourceID: "score-it", TargetID: "gate"},
				{SourceID: "gate", TargetID: "file-it"},
			},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Equal(t, "listen-here", graph.TriggerNodeID)
		assert.Equal(t, "score-it", graph.AnalysisNodeID)
		assert.Equal(t, "file-it", graph.CreateNodeID)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("a graph without a score step is not healthy", func(t *testing.T) {
		spec := models.LiveCanvasSpec{
			Nodes: []models.Node{
				triggerNode(intakeTriggerNodeID, "github.onIssue"),
				componentNode(intakeCreateNodeID, intakeCreateComponent),
			},
			Edges: []models.Edge{{SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID}},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Empty(t, graph.AnalysisNodeID)
		assert.False(t, graph.Healthy(spec.Edges))
	})

	t.Run("a disconnected graph is not healthy", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues, DefaultIntakeConfidencePct)
		spec.Edges = nil

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.False(t, graph.Healthy(spec.Edges))
	})

	t.Run("extra nodes on the path keep the graph healthy", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues, DefaultIntakeConfidencePct)
		spec.Nodes = append(spec.Nodes, componentNode("label-filter", "filter"))
		spec.Edges = []models.Edge{
			{SourceID: intakeTriggerNodeID, TargetID: "label-filter"},
			{SourceID: "label-filter", TargetID: intakeAnalysisNodeID},
			{SourceID: intakeAnalysisNodeID, TargetID: intakeThresholdNodeID},
			{SourceID: intakeThresholdNodeID, TargetID: intakeCreateNodeID},
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.True(t, graph.Healthy(spec.Edges))
	})

	t.Run("a hand-edited threshold falls back to the default", func(t *testing.T) {
		spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceGitHubIssues, DefaultIntakeConfidencePct)
		for i := range spec.Nodes {
			if spec.Nodes[i].ID == intakeThresholdNodeID {
				spec.Nodes[i].Configuration = map[string]any{"expression": "$.result == 'ship it'"}
			}
		}

		graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, spec)
		assert.Equal(t, DefaultIntakeConfidencePct, graph.ConfidencePct)
	})
}

func intakeSpecFromTemplate(t *testing.T, source string, confidencePct int) models.LiveCanvasSpec {
	t.Helper()

	canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: source, ConfidencePct: confidencePct})
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
