package contexts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRunnerEmitConstantsAlignedWithRunnerComponent(t *testing.T) {
	assert.Equal(t, runner.RunnerFinishedEventType, runnerFinishedPayloadType)
	assert.Equal(t, runner.FailedOutputChannel, runnerFailedOutputChannelName)
}

func Test__ExecutionStateContext__Emit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNodeID,
				Name:   componentNodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	t.Run("rejects large payload", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), execution, nil)
		largePayload := strings.Repeat("a", DefaultMaxPayloadSize+100)

		err := ctx.Emit("default", "test.payload", []any{largePayload})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event payload too large")
		support.VerifyCanvasNodeEventsCount(t, canvas.ID, componentNodeID, 0)
	})

	t.Run("uses callback", func(t *testing.T) {
		newEvents := []models.CanvasEvent{}
		onNewEvents := func(events []models.CanvasEvent) {
			newEvents = append(newEvents, events...)
		}

		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), execution, onNewEvents)
		err := ctx.Emit("default", "test.payload", []any{map[string]any{"n": 1}})
		require.NoError(t, err)
		assert.Len(t, newEvents, 1)
	})

	t.Run("runner finished on failed channel marks canvas execution failed", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), execution, nil)
		err := ctx.Emit(runner.FailedOutputChannel, runner.RunnerFinishedEventType, []any{map[string]any{"error": "interactive runner needs bash"}})
		require.NoError(t, err)

		var reloaded models.CanvasNodeExecution
		require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&reloaded).Error)
		assert.Equal(t, models.CanvasNodeExecutionResultFailed, reloaded.Result)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, reloaded.ResultReason)
		assert.Equal(t, "interactive runner needs bash", reloaded.ResultMessage)

		var eventCount int64
		require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("execution_id = ?", execution.ID).Count(&eventCount).Error)
		require.EqualValues(t, 1, eventCount)
	})

	t.Run("runner finished on passed channel marks canvas execution passed", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), execution, nil)
		require.NoError(t, ctx.Emit(runner.PassedOutputChannel, runner.RunnerFinishedEventType, []any{map[string]any{"status": "succeeded", "exit_code": 0}}))

		var reloaded models.CanvasNodeExecution
		require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&reloaded).Error)
		assert.Equal(t, models.CanvasNodeExecutionResultPassed, reloaded.Result)
	})
}
