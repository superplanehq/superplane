package contexts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

// stubAction is a minimal core.Action for Emit-routing tests. Only
// OutputChannels is implemented; all other methods panic if ever called
// via the embedded nil interface. This avoids re-stubbing the full
// Action interface for each test.
type stubAction struct {
	core.Action // embedded nil interface — unused methods panic
	channels    []core.OutputChannel
}

func (s *stubAction) OutputChannels(any) []core.OutputChannel { return s.channels }

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

	failureAction := &stubAction{channels: []core.OutputChannel{
		{Name: "success", Label: "Success"},
		{Name: "failure", Label: "Failure"},
	}}

	t.Run("rejects large payload", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), nil, execution, nil)
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

		ctx := NewExecutionStateContext(database.Conn(), nil, execution, onNewEvents)
		err := ctx.Emit("default", "test.payload", []any{map[string]any{"n": 1}})
		require.NoError(t, err)
		assert.Len(t, newEvents, 1)
	})

	// Regression test for #4284: emitting to a channel whose Label is a
	// known failure word must still create the channel event (so downstream
	// nodes fire) but mark the execution row as failed.
	t.Run("failure channel finishes execution as failed and still emits event", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), failureAction, execution, nil)
		err := ctx.Emit("failure", "test.payload", []any{map[string]any{"err": "boom"}})
		require.NoError(t, err)

		var reloaded models.CanvasNodeExecution
		require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&reloaded).Error)
		assert.Equal(t, models.CanvasNodeExecutionStateFinished, reloaded.State)
		assert.Equal(t, models.CanvasNodeExecutionResultFailed, reloaded.Result)
		assert.NotEmpty(t, reloaded.ResultReason)
		assert.NotEmpty(t, reloaded.ResultMessage)

		// The routed event must still be persisted for this execution.
		// Scope the count to the execution ID so it's independent of
		// events left behind by sibling subtests on the same node.
		var eventsForExecution int64
		require.NoError(t, database.Conn().
			Model(&models.CanvasEvent{}).
			Where("execution_id = ?", execution.ID).
			Where("channel = ?", "failure").
			Count(&eventsForExecution).Error)
		assert.Equal(t, int64(1), eventsForExecution)
	})

	// Emitting to a non-failure channel must continue to mark the
	// execution as passed — preserve existing behavior.
	t.Run("success channel still records passed", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)

		ctx := NewExecutionStateContext(database.Conn(), failureAction, execution, nil)
		err := ctx.Emit("success", "test.payload", []any{map[string]any{"ok": true}})
		require.NoError(t, err)

		var reloaded models.CanvasNodeExecution
		require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&reloaded).Error)
		assert.Equal(t, models.CanvasNodeExecutionStateFinished, reloaded.State)
		assert.Equal(t, models.CanvasNodeExecutionResultPassed, reloaded.Result)
	})

	// A child execution routing to its failure channel must NOT bubble the
	// failed status up to the parent: the emitted event is what drives
	// parent aggregation via completeParentExecutionIfNeeded, so
	// propagating would double-count. FailInTransaction (no-emit path)
	// still bubbles; only the emit-with-failure path skips it.
	t.Run("failure channel does not bubble failed status to parent", func(t *testing.T) {
		rootData := map[string]any{"root": "event"}
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, rootData)

		parent := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, nil)
		child := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID, &parent.ID)

		ctx := NewExecutionStateContext(database.Conn(), failureAction, child, nil)
		require.NoError(t, ctx.Emit("failure", "test.payload", []any{map[string]any{"err": "boom"}}))

		var reloadedChild models.CanvasNodeExecution
		require.NoError(t, database.Conn().Where("id = ?", child.ID).First(&reloadedChild).Error)
		assert.Equal(t, models.CanvasNodeExecutionResultFailed, reloadedChild.Result, "child must be recorded as failed")

		var reloadedParent models.CanvasNodeExecution
		require.NoError(t, database.Conn().Where("id = ?", parent.ID).First(&reloadedParent).Error)
		assert.Equal(t, models.CanvasNodeExecutionStatePending, reloadedParent.State, "parent state must be untouched")
		assert.Empty(t, reloadedParent.Result, "parent result must not be set by child routed-failure")
	})
}
