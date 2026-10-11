package factories

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__linearCompletionStateID(t *testing.T) {
	states := []linear.WorkflowState{
		{ID: "todo", Name: "Todo", Type: "unstarted"},
		{ID: "review", Name: "In Review", Type: "started"},
		{ID: "shipped", Name: "Shipped", Type: "completed"},
		{ID: "done", Name: "Done", Type: "completed"},
	}

	id, ok := linearCompletionStateID(states, "in review")
	assert.True(t, ok)
	assert.Equal(t, "review", id)

	id, ok = linearCompletionStateID(states, "")
	assert.True(t, ok)
	assert.Equal(t, "done", id)

	id, ok = linearCompletionStateID(states[:3], "")
	assert.True(t, ok)
	assert.Equal(t, "shipped", id)

	_, ok = linearCompletionStateID(states, "Missing")
	assert.False(t, ok)

	_, ok = linearCompletionStateID(states[:2], "")
	assert.False(t, ok)
}

func Test__linearCloseRetryable(t *testing.T) {
	assert.False(t, linearCloseRetryable(nil))
	assert.True(t, linearCloseRetryable(errors.New("request got 500 code: boom")))
	assert.True(t, linearCloseRetryable(errors.New("request got 429 code: slow down")))
	assert.True(t, linearCloseRetryable(errors.New("request got 408 code: timeout")))
	assert.True(t, linearCloseRetryable(errors.New("error executing request: dial timeout")))
	assert.False(t, linearCloseRetryable(errors.New("request got 404 code: not found")))
	assert.False(t, linearCloseRetryable(errors.New("request got 401 code: unauthorized")))
	assert.False(t, linearCloseRetryable(errors.New("linear API error: invalid state")))
}

func Test__linearCompletionSettingsFromMetadata(t *testing.T) {
	settings := linearCompletionSettingsFromMetadata(map[string]any{
		intakeMetadataLinearMoveOnComplete:  false,
		intakeMetadataLinearCompletionState: "  In Review ",
	}, defaultLinearIntakeSettings())
	assert.False(t, settings.LinearMoveOnComplete)
	assert.Equal(t, "In Review", settings.LinearCompletionState)

	merged := mergeLinearCompletionMetadata(map[string]any{"other": true}, settings)
	assert.Equal(t, false, merged[intakeMetadataLinearMoveOnComplete])
	assert.Equal(t, "In Review", merged[intakeMetadataLinearCompletionState])
	assert.Equal(t, true, merged["other"])
}

func Test__intakeSettingsFromGraph__LinearCompletion(t *testing.T) {
	graph := intakeGraph{TriggerNodeID: intakeTriggerNodeID}
	spec := models.LiveCanvasSpec{Nodes: []models.Node{{
		ID:            intakeTriggerNodeID,
		Configuration: map[string]any{"projects": []any{"project-1"}},
		Metadata: map[string]any{
			intakeMetadataLinearMoveOnComplete:  false,
			intakeMetadataLinearCompletionState: "In Review",
		},
	}}}

	settings := intakeSettingsFromGraph(models.FactoryIntakeSourceLinearIssues, graph, spec)
	assert.False(t, settings.LinearMoveOnComplete)
	assert.Equal(t, "In Review", settings.LinearCompletionState)
	assert.Equal(t, []string{"project-1"}, settings.LinearProjectIDs)
}
