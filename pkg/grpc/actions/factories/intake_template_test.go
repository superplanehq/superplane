package factories

import (
	"strconv"
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func Test__BuildIntakeCanvas(t *testing.T) {
	t.Run("each source listens with its own trigger", func(t *testing.T) {
		for source, expected := range map[string]string{
			models.FactoryIntakeSourceGitHubIssues:       "github.onIssue",
			models.FactoryIntakeSourceSentryExceptions:   "sentry.onIssue",
			models.FactoryIntakeSourcePagerDutyIncidents: "pagerduty.onIncident",
		} {
			canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: source, ConfidencePct: DefaultIntakeConfidencePct})
			require.NoError(t, err)

			trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
			assert.Equal(t, expected, trigger.Component)
			assert.Equal(t, yaml.NodeTypeTrigger, trigger.Type)
		}
	})

	t.Run("the item flows from the trigger to the work order", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeAnalysisNodeID},
			{Channel: "passed", SourceID: intakeAnalysisNodeID, TargetID: intakeThresholdNodeID},
			{Channel: "true", SourceID: intakeThresholdNodeID, TargetID: intakeCreateNodeID},
			{Channel: "default", SourceID: intakeCreateNodeID, TargetID: intakeReportConfidenceNodeID},
		}, canvas.Spec.Edges)
	})

	t.Run("the created work order receives the intake confidence score", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		report := findSpecNode(t, canvas, intakeReportConfidenceNodeID)
		assert.Equal(t, intakeReportConfidenceComponent, report.Component)
		assert.Equal(t, "{{ $[\"Create Work Order\"].data.workOrder.id }}", report.Configuration["orderId"])
		assert.Equal(t, "confidence", report.Configuration["checkKey"])
		assert.Equal(t, "Confidence score", report.Configuration["name"])
		assert.Equal(t, "5", report.Configuration["maxScore"])
		assert.Equal(t, "fraction", report.Configuration["format"])
	})

	t.Run("the score rounds onto the meter scale like the UI does", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		report := findSpecNode(t, canvas, intakeReportConfidenceNodeID)
		assert.Equal(
			t,
			`{{ int(round(int($["Analyze intake"].data.result.result) / 20.0)) }}`,
			report.Configuration["score"],
		)

		expression := report.Configuration["score"].(string)
		for pct, expected := range map[int]int{0: 0, 49: 2, 50: 3, 69: 3, 70: 4, 89: 4, 90: 5, 100: 5} {
			assert.Equal(t, expected, evaluateIntakeConfidenceScore(t, expression, pct), "confidence %d%%", pct)
		}
	})

	t.Run("the check level follows the meter bands", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		report := findSpecNode(t, canvas, intakeReportConfidenceNodeID)
		assert.Equal(t, "higherIsBetter", report.Configuration["direction"])
		// High starts at 4, so 3 is caution (Medium) and 2 and below is
		// critical (Low).
		assert.Equal(t, float64(3), report.Configuration["cautionAt"])
		assert.Equal(t, float64(2), report.Configuration["criticalAt"])
	})

	t.Run("the analysis runner asks for a machine", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		analysis := findSpecNode(t, canvas, intakeAnalysisNodeID)
		assert.Equal(t, runner.MachineTypeE1LargeAMD64, analysis.Configuration["machineType"])
	})

	t.Run("the analysis runner authenticates with the workspace agent", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{
			Source:        models.FactoryIntakeSourceGitHubIssues,
			ConfidencePct: DefaultIntakeConfidencePct,
			Agent: &intakeAgent{
				Component: "runnerCodex",
				Credentials: map[string]any{
					"source":      runner.CredentialsSourceIntegration,
					"integration": map[string]any{"name": "acme-openai"},
				},
			},
		})
		require.NoError(t, err)

		analysis := findSpecNode(t, canvas, intakeAnalysisNodeID)
		assert.Equal(t, "runnerCodex", analysis.Component)
		assert.Equal(t, map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": "acme-openai"},
		}, analysis.Configuration["credentials"])
		// Codex reads its own default model, so the node leaves the model out.
		assert.NotContains(t, analysis.Configuration, "model")
	})

	t.Run("a hosted agent names the model it runs", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{
			Source:        models.FactoryIntakeSourceGitHubIssues,
			ConfidencePct: DefaultIntakeConfidencePct,
			Agent: &intakeAgent{
				Component:   "runnerClaudeCode",
				Credentials: map[string]any{"source": runner.CredentialsSourceHosted},
				Model:       "claude-sonnet-4-6",
			},
		})
		require.NoError(t, err)

		analysis := findSpecNode(t, canvas, intakeAnalysisNodeID)
		assert.Equal(t, "claude-sonnet-4-6", analysis.Configuration["model"])
	})

	t.Run("an intake without an agent leaves the credentials to the user", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		analysis := findSpecNode(t, canvas, intakeAnalysisNodeID)
		assert.Equal(t, intakeAgentSpecs[0].component, analysis.Component)
		assert.NotContains(t, analysis.Configuration, "credentials")
	})

	t.Run("the threshold gates on the analysis score", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: 80})
		require.NoError(t, err)

		threshold := findSpecNode(t, canvas, intakeThresholdNodeID)
		assert.Equal(t, `int($["Analyze intake"].data.result.result) >= 80`, threshold.Configuration["expression"])
	})

	t.Run("the threshold reads the score the analysis runner reports", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: 70})
		require.NoError(t, err)

		threshold := findSpecNode(t, canvas, intakeThresholdNodeID)
		expression := threshold.Configuration["expression"].(string)
		assert.False(t, evaluateIntakeThreshold(t, expression, 69))
		assert.True(t, evaluateIntakeThreshold(t, expression, 70))
	})

	t.Run("every action node works on a whole batch at once", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		for _, node := range canvas.Spec.Nodes {
			// A trigger has no queue of its own, and the parser rejects a
			// concurrency spec on it.
			if node.Type == yaml.NodeTypeTrigger {
				assert.Nilf(t, node.Concurrency, "trigger %s caps its concurrency", node.ID)
				continue
			}

			require.NotNilf(t, node.Concurrency, "node %s has no concurrency", node.ID)
			require.NotNilf(t, node.Concurrency.Max, "node %s has no concurrency max", node.ID)
			assert.Equalf(t, intakeConcurrencyMax, *node.Concurrency.Max, "node %s", node.ID)
		}
	})

	t.Run("confidence outside the scale is clamped", func(t *testing.T) {
		for confidence, expected := range map[int]int{-20: 0, 0: 0, 65: 65, 100: 100, 140: 100} {
			canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: confidence})
			require.NoError(t, err)

			threshold := findSpecNode(t, canvas, intakeThresholdNodeID)
			parsed, ok := intakeConfidenceFromExpression(threshold.Configuration["expression"].(string))
			require.True(t, ok)
			assert.Equal(t, expected, parsed)
		}
	})

	t.Run("a given name wins over the source default", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Name: "Backlog triage", ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)
		assert.Equal(t, "Backlog triage", canvas.Metadata.Name)

		canvas, err = buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, Name: "   ", ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)
		assert.Equal(t, "GitHub issues", canvas.Metadata.Name)
	})

	t.Run("an unknown source has no graph", func(t *testing.T) {
		_, err := buildIntakeCanvas(intakeCanvasRequest{Source: "linear-issues", ConfidencePct: DefaultIntakeConfidencePct})
		assert.ErrorIs(t, err, models.ErrFactoryIntakeSourceInvalid)
	})

	t.Run("the binding tells the trigger what to listen on", func(t *testing.T) {
		binding := &intakeBinding{
			Integration:   &yaml.IntegrationRef{ID: "integration-1", Name: "acme-github"},
			Configuration: map[string]any{"repository": "acme/backlog"},
		}
		canvas, err := buildIntakeCanvas(intakeCanvasRequest{
			Source:        models.FactoryIntakeSourceGitHubIssues,
			ConfidencePct: DefaultIntakeConfidencePct,
			Binding:       binding,
		})
		require.NoError(t, err)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Equal(t, binding.Integration, trigger.Integration)
		assert.Equal(t, "acme/backlog", trigger.Configuration["repository"])
		// The binding names the resource; the template still picks the events.
		assert.Equal(t, []any{"opened"}, trigger.Configuration["actions"])
	})

	t.Run("a binding does not leak into the next intake", func(t *testing.T) {
		_, err := buildIntakeCanvas(intakeCanvasRequest{
			Source:        models.FactoryIntakeSourceGitHubIssues,
			ConfidencePct: DefaultIntakeConfidencePct,
			Binding: &intakeBinding{
				Configuration: map[string]any{"repository": "acme/backlog"},
			},
		})
		require.NoError(t, err)

		canvas, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceGitHubIssues, ConfidencePct: DefaultIntakeConfidencePct})
		require.NoError(t, err)

		trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
		assert.Nil(t, trigger.Integration)
		assert.NotContains(t, trigger.Configuration, "repository")
	})
}

func Test__IntakeConfidenceFromExpression(t *testing.T) {
	t.Run("reads back a generated expression", func(t *testing.T) {
		confidence, ok := intakeConfidenceFromExpression(intakeThresholdExpression(42))
		require.True(t, ok)
		assert.Equal(t, 42, confidence)
	})

	t.Run("reports failure for a hand-written expression", func(t *testing.T) {
		_, ok := intakeConfidenceFromExpression("$.result == 'ship it'")
		assert.False(t, ok)
	})
}

// evaluateIntakeConfidenceScore runs the generated score expression against an
// analysis result of pct, so the band edges are checked with the same engine
// that resolves node configuration at run time.
func evaluateIntakeConfidenceScore(t *testing.T, expression string, pct int) int {
	t.Helper()

	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expression, "{{"), "}}"))
	output := evaluateIntakeExpression(t, inner, pct)

	score, ok := output.(int)
	require.Truef(t, ok, "expression returned %T, want int", output)

	return score
}

// evaluateIntakeThreshold runs the generated threshold expression. The if
// component requires a boolean, so a wrong result path fails here instead of at
// run time.
func evaluateIntakeThreshold(t *testing.T, expression string, pct int) bool {
	t.Helper()

	output := evaluateIntakeExpression(t, expression, pct)

	matches, ok := output.(bool)
	require.Truef(t, ok, "expression returned %T, want bool", output)

	return matches
}

// evaluateIntakeExpression resolves an expression against the event the
// analysis runner emits when it finishes, so the generated result path is
// checked against the shape the graph actually receives.
func evaluateIntakeExpression(t *testing.T, expression string, pct int) any {
	t.Helper()

	analysis := map[string]any{
		"type": intakeAgentSpecs[0].component + ".finished",
		"data": map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result": map[string]any{
				"type":   "result",
				"result": strconv.Itoa(pct),
			},
		},
	}

	output, err := expr.Eval(expression, map[string]any{
		"$": map[string]any{intakeAnalysisNodeName: analysis},
	})
	require.NoError(t, err)

	return output
}

func findSpecNode(t *testing.T, canvas *yaml.Canvas, nodeID string) yaml.Node {
	t.Helper()

	for _, node := range canvas.Spec.Nodes {
		if node.ID == nodeID {
			return node
		}
	}

	require.Failf(t, "node not found", "canvas has no node %q", nodeID)
	return yaml.Node{}
}
