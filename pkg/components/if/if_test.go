package ifp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestIf_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		inputData       any
		expectedChannel string
	}{
		{
			name:            "if with true condition emits empty event",
			configuration:   map[string]any{"expression": "true"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "true",
		},
		{
			name:            "if with false condition emits empty event",
			configuration:   map[string]any{"expression": "false"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "false",
		},
		{
			name:            "if with complex true condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'value'"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "true",
		},
		{
			name:            "if with complex false condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'different'"},
			inputData:       map[string]any{"test": "value"},
			expectedChannel: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ifComponent := &If{}

			stateCtx := &contexts.ExecutionStateContext{}

			ctx := core.ExecutionContext{
				Data:                  tt.inputData,
				Configuration:         tt.configuration,
				ExecutionStateContext: stateCtx,
			}

			err := ifComponent.Execute(ctx)

			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)
			assert.Len(t, stateCtx.Outputs, 1)
			assert.Equal(t, tt.expectedChannel, stateCtx.Outputs[0].Channel)
			assert.Len(t, stateCtx.Outputs[0].Payloads, 1)
			assert.Equal(t, "if.executed", stateCtx.Outputs[0].Payloads[0].Type)
			assert.Equal(t, make(map[string]any), stateCtx.Outputs[0].Payloads[0].Data)
		})
	}
}

func TestIf_Execute_InvalidExpression_ShouldReturnError(t *testing.T) {
	ifComponent := &If{}

	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "invalid expression syntax +++"},
		ExecutionStateContext: stateCtx,
	}

	err := ifComponent.Execute(ctx)
	assert.Error(t, err)

}

func TestIf_Execute_NonBooleanResult_ShouldReturnError(t *testing.T) {
	ifComponent := &If{}

	stateCtx := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "$.test"},
		ExecutionStateContext: stateCtx,
	}

	err := ifComponent.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression must evaluate to boolean")
}

func TestIf_Execute_BothTrueAndFalsePathsEmitEmpty(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		expectedChannel string
	}{
		{
			name:            "true condition previously went to true channel, now emits empty",
			configuration:   map[string]any{"expression": "true"},
			expectedChannel: "true",
		},
		{
			name:            "false condition previously went to false channel, now emits empty",
			configuration:   map[string]any{"expression": "false"},
			expectedChannel: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ifComponent := &If{}

			stateCtx := &contexts.ExecutionStateContext{}

			ctx := core.ExecutionContext{
				Data:                  map[string]any{"test": "value"},
				Configuration:         tt.configuration,
				ExecutionStateContext: stateCtx,
			}

			err := ifComponent.Execute(ctx)
			assert.NoError(t, err)
			assert.True(t, stateCtx.Passed)
			assert.True(t, stateCtx.Finished)
			assert.Len(t, stateCtx.Outputs, 1)
			assert.Equal(t, tt.expectedChannel, stateCtx.Outputs[0].Channel)
			assert.Len(t, stateCtx.Outputs[0].Payloads, 1)
			assert.Equal(t, "if.executed", stateCtx.Outputs[0].Payloads[0].Type)
			assert.Equal(t, make(map[string]any), stateCtx.Outputs[0].Payloads[0].Data)
		})
	}
}
