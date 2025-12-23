package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestFilter_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name                 string
		configuration        map[string]any
		inputData            any
		expectedOutputsCount int
		expectedChannel      string
	}{
		{
			name:                 "filter with true condition emits empty event",
			configuration:        map[string]any{"expression": "true"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 1,
			expectedChannel:      "default",
		},
		{
			name:                 "filter with false condition emits empty event",
			configuration:        map[string]any{"expression": "false"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 0,
			expectedChannel:      "",
		},
		{
			name:                 "filter with complex true condition emits empty event",
			configuration:        map[string]any{"expression": "$.test == 'value'"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 1,
			expectedChannel:      "default",
		},
		{
			name:                 "filter with complex false condition emits empty event",
			configuration:        map[string]any{"expression": "$.test == 'different'"},
			inputData:            map[string]any{"test": "value"},
			expectedOutputsCount: 0,
			expectedChannel:      "",
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
			assert.Len(t, stateCtx.Outputs, tt.expectedOutputsCount)
			if tt.expectedOutputsCount > 0 {
				assert.Equal(t, tt.expectedChannel, stateCtx.Outputs[0].Channel)
				assert.Len(t, stateCtx.Outputs[0].Payloads, 1)
				assert.Equal(t, "filter.executed", stateCtx.Outputs[0].Payloads[0].Type)
				assert.Equal(t, make(map[string]any), stateCtx.Outputs[0].Payloads[0].Data)
			}
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
