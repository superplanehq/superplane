package ifp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/superplanehq/superplane/pkg/core"
)

type MockExecutionStateContext struct {
	mock.Mock
}

func (m *MockExecutionStateContext) Pass(outputs map[string][]any) error {
	args := m.Called(outputs)
	return args.Error(0)
}

func (m *MockExecutionStateContext) Fail(reason, message string) error {
	args := m.Called(reason, message)
	return args.Error(0)
}

func (m *MockExecutionStateContext) IsFinished() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestIf_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		inputData       any
		expectedOutputs map[string][]any
	}{
		{
			name:            "if with true condition emits empty event",
			configuration:   map[string]any{"expression": "true"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{"true": {make(map[string]any)}},
		},
		{
			name:            "if with false condition emits empty event",
			configuration:   map[string]any{"expression": "false"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{"false": {make(map[string]any)}},
		},
		{
			name:            "if with complex true condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'value'"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{"true": {make(map[string]any)}},
		},
		{
			name:            "if with complex false condition emits empty event",
			configuration:   map[string]any{"expression": "$.test == 'different'"},
			inputData:       map[string]any{"test": "value"},
			expectedOutputs: map[string][]any{"false": {make(map[string]any)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ifComponent := &If{}

			mockExecStateCtx := &MockExecutionStateContext{}

			mockExecStateCtx.On("Pass", tt.expectedOutputs).Return(nil)

			ctx := core.ExecutionContext{
				Data:                  tt.inputData,
				Configuration:         tt.configuration,
				ExecutionStateContext: mockExecStateCtx,
			}

			err := ifComponent.Execute(ctx)

			assert.NoError(t, err)

			mockExecStateCtx.AssertExpectations(t)
		})
	}
}

func TestIf_Execute_InvalidExpression_ShouldReturnError(t *testing.T) {
	ifComponent := &If{}

	mockExecStateCtx := &MockExecutionStateContext{}

	ctx := core.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "invalid expression syntax +++"},
		ExecutionStateContext: mockExecStateCtx,
	}

	err := ifComponent.Execute(ctx)
	assert.Error(t, err)
	assert.Error(t, err)

}

func TestIf_Execute_NonBooleanResult_ShouldReturnError(t *testing.T) {
	ifComponent := &If{}

	mockExecStateCtx := &MockExecutionStateContext{}

	ctx := core.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "$.test"},
		ExecutionStateContext: mockExecStateCtx,
	}

	err := ifComponent.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression must evaluate to boolean")
}

func TestIf_Execute_BothTrueAndFalsePathsEmitEmpty(t *testing.T) {
	tests := []struct {
		name            string
		configuration   map[string]any
		expectedOutputs map[string][]any
	}{
		{
			name:            "true condition previously went to true channel, now emits empty",
			configuration:   map[string]any{"expression": "true"},
			expectedOutputs: map[string][]any{"true": {make(map[string]any)}},
		},
		{
			name:            "false condition previously went to false channel, now emits empty",
			configuration:   map[string]any{"expression": "false"},
			expectedOutputs: map[string][]any{"false": {make(map[string]any)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ifComponent := &If{}

			mockExecStateCtx := &MockExecutionStateContext{}

			mockExecStateCtx.On("Pass", tt.expectedOutputs).Return(nil)

			ctx := core.ExecutionContext{
				Data:                  map[string]any{"test": "value"},
				Configuration:         tt.configuration,
				ExecutionStateContext: mockExecStateCtx,
			}

			err := ifComponent.Execute(ctx)
			assert.NoError(t, err)
			mockExecStateCtx.AssertExpectations(t)
		})
	}
}
