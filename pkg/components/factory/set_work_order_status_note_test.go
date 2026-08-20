package factory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSetWorkOrderStatusNote_Execute(t *testing.T) {
	component := &SetWorkOrderStatusNote{}
	stored := &core.WorkOrderStatusNote{
		WorkOrderID: "wo-1",
		Key:         "pr-closure",
		Kind:        "info",
		Headline:    "Review the pull request",
		Body:        "When PR #42 merges, this work order completes automatically.",
		CtaLabel:    "Review PR #42",
		CtaURL:      "https://github.com/acme/app/pull/42",
	}

	t.Run("sets the note and emits workOrder.statusNoteSet", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{setStatusNoteResult: stored}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"noteKey":  "pr-closure",
				"headline": "Review the pull request",
				"body":     "When PR #42 merges, this work order completes automatically.",
				"ctaLabel": "Review PR #42",
				"ctaUrl":   "https://github.com/acme/app/pull/42",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.setStatusNoteCalls)
		assert.Equal(t, "wo-1", factoryCtx.setStatusNoteParams.OrderID)
		assert.Equal(t, "pr-closure", factoryCtx.setStatusNoteParams.NoteKey)
		assert.Equal(t, "Review the pull request", factoryCtx.setStatusNoteParams.Headline)
		assert.Equal(t, "Review PR #42", factoryCtx.setStatusNoteParams.CtaLabel)
		assert.Equal(t, "https://github.com/acme/app/pull/42", factoryCtx.setStatusNoteParams.CtaURL)
		assert.False(t, factoryCtx.setStatusNoteParams.ShowOnlyWhenWaiting)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "workOrder.statusNoteSet", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	t.Run("propagates factory context errors without emitting", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{setStatusNoteErr: errors.New("work order must be open")}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":  "wo-1",
				"headline": "Review the pull request",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work order must be open")
		assert.Empty(t, stateCtx.Payloads)
	})

	t.Run("passes showOnlyWhenWaiting through to the factory context", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{setStatusNoteResult: stored}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"orderId":             "wo-1",
				"noteKey":             "queue-slot",
				"headline":            "Waiting for a slot",
				"showOnlyWhenWaiting": true,
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.True(t, factoryCtx.setStatusNoteParams.ShowOnlyWhenWaiting)
	})
}
