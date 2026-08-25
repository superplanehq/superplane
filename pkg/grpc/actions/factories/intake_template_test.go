package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			canvas, err := buildIntakeCanvas(source, "", DefaultIntakeConfidencePct)
			require.NoError(t, err)

			trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
			assert.Equal(t, expected, trigger.Component)
			assert.Equal(t, yaml.NodeTypeTrigger, trigger.Type)
		}
	})

	t.Run("the item flows from the trigger to the work order", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(models.FactoryIntakeSourceGitHubIssues, "", DefaultIntakeConfidencePct)
		require.NoError(t, err)

		assert.Equal(t, []yaml.Edge{
			{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeAnalysisNodeID},
			{Channel: "passed", SourceID: intakeAnalysisNodeID, TargetID: intakeThresholdNodeID},
			{Channel: "true", SourceID: intakeThresholdNodeID, TargetID: intakeCreateNodeID},
		}, canvas.Spec.Edges)
	})

	t.Run("the threshold gates on the analysis score", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(models.FactoryIntakeSourceGitHubIssues, "", 80)
		require.NoError(t, err)

		threshold := findSpecNode(t, canvas, intakeThresholdNodeID)
		assert.Equal(t, `int($["Analyze intake"].data[0].result.result) >= 80`, threshold.Configuration["expression"])
	})

	t.Run("confidence outside the scale is clamped", func(t *testing.T) {
		for confidence, expected := range map[int]int{-20: 0, 0: 0, 65: 65, 100: 100, 140: 100} {
			canvas, err := buildIntakeCanvas(models.FactoryIntakeSourceGitHubIssues, "", confidence)
			require.NoError(t, err)

			threshold := findSpecNode(t, canvas, intakeThresholdNodeID)
			parsed, ok := intakeConfidenceFromExpression(threshold.Configuration["expression"].(string))
			require.True(t, ok)
			assert.Equal(t, expected, parsed)
		}
	})

	t.Run("a given name wins over the source default", func(t *testing.T) {
		canvas, err := buildIntakeCanvas(models.FactoryIntakeSourceGitHubIssues, "Backlog triage", DefaultIntakeConfidencePct)
		require.NoError(t, err)
		assert.Equal(t, "Backlog triage", canvas.Metadata.Name)

		canvas, err = buildIntakeCanvas(models.FactoryIntakeSourceGitHubIssues, "   ", DefaultIntakeConfidencePct)
		require.NoError(t, err)
		assert.Equal(t, "GitHub issues", canvas.Metadata.Name)
	})

	t.Run("an unknown source has no graph", func(t *testing.T) {
		_, err := buildIntakeCanvas("linear-issues", "", DefaultIntakeConfidencePct)
		assert.ErrorIs(t, err, models.ErrFactoryIntakeSourceInvalid)
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
