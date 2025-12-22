package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestFilter_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		inputData       any
		expectedOutputs map[string][]any
	}{
		{
			name:            "filter with true condition emits empty event",
			configuration:   map[string]any{"expression": "true"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{"default": {make(map[string]any)}},
		},
		{
			name:            "filter with false condition emits empty event",
			configuration:   map[string]any{"expression": "false"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{},
		},
		{
			name:            "filter with complex true condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'value'"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{"default": {make(map[string]any)}},
		},
		{
			name:            "filter with complex false condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'different'"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			filter := &Filter{}

			stateCtx := &contexts.ExecutionStateContext{}

			ctx := core.ExecutionContext{
				Data:                  tt.inputData,
				Configuration:         tt.configuration,
				ExecutionStateContext: stateCtx,
			}

			err := filter.Execute(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)
			assert.Equal(t, tt.expectedOutputs, stateCtx.Outputs)
		})
	}
}

func TestFilter_Execute_InvalidExpression_ShouldReturnError(t *testing.T) {
	filter := &Filter{}

	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "invalid expression syntax +++"},
		ExecutionStateContext: stateCtx,
	}

	err := filter.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression compilation failed")
}

func TestFilter_Execute_NonBooleanResult_ShouldReturnError(t *testing.T) {
	filter := &Filter{}

	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "$.test"},
		ExecutionStateContext: stateCtx,
	}

	err := filter.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression must evaluate to boolean")
}
