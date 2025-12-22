package noop

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestNoop_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name      string
		inputData any
	}{
		{
			name:      "noop with simple data emits empty event",
			inputData: map[string]any{"test": "value"},
		},
		{
			name:      "noop with complex data emits empty event",
			inputData: map[string]any{"nested": map[string]any{"key": "value"}, "array": []any{1, 2, 3}},
		},
		{
			name:      "noop with nil data emits empty event",
			inputData: nil,
		},
		{
			name:      "noop with string data emits empty event",
			inputData: "test string",
		},
		{
			name:      "noop with numeric data emits empty event",
			inputData: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noop := &NoOp{}

			stateCtx := &contexts.ExecutionStateContext{}

			ctx := core.ExecutionContext{
				Data:                  tt.inputData,
				ExecutionStateContext: stateCtx,
			}

			err := noop.Execute(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)
			assert.Equal(t, map[string][]any{"default": {make(map[string]any)}}, stateCtx.Outputs)
		})
	}
}

func TestNoop_Execute_AlwaysEmitsEmpty(t *testing.T) {
	t.Run("noop should never pass through original data", func(t *testing.T) {
		noop := &NoOp{}
		stateCtx := &contexts.ExecutionStateContext{}

		originalData := map[string]any{
			"important": "data",
			"that":      "should not be passed through",
		}

		ctx := core.ExecutionContext{
			Data:                  originalData,
			ExecutionStateContext: stateCtx,
		}

		err := noop.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)

		// Verify that the output is an empty map, not the original data
		expectedOutputs := map[string][]any{"default": {make(map[string]any)}}
		assert.Equal(t, expectedOutputs, stateCtx.Outputs)
		assert.NotEqual(t, map[string][]any{core.DefaultOutputChannel.Name: {originalData}}, stateCtx.Outputs)
	})
}

func TestNoop_Configuration_ShouldReturnEmptyConfig(t *testing.T) {
	noop := &NoOp{}
	config := noop.Configuration()

	assert.Empty(t, config)
}

func TestNoop_Actions_ShouldReturnEmptyActions(t *testing.T) {
	noop := &NoOp{}
	actions := noop.Actions()

	assert.Empty(t, actions)
}

func TestNoop_BasicProperties_ShouldReturnCorrectValues(t *testing.T) {
	noop := &NoOp{}

	assert.Equal(t, "noop", noop.Name())
	assert.Equal(t, "No Operation", noop.Label())
	assert.Contains(t, noop.Description(), "pass events through")
	assert.Equal(t, "circle-off", noop.Icon())
	assert.Equal(t, "blue", noop.Color())
}
