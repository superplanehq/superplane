package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/superplanehq/superplane/pkg/components"
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

func TestFilter_Execute_EmitsEmptyEvents(t *testing.T) {
	tests := []struct {
		name          string
		configuration map[string]any
		inputData     any
	}{
		{
			name:          "filter with true condition emits empty event",
			configuration: map[string]any{"expression": "true"},
			inputData:     map[string]any{"test": "value"},
		},
		{
			name:          "filter with false condition emits empty event",
			configuration: map[string]any{"expression": "false"},
			inputData:     map[string]any{"test": "value"},
		},
		{
			name:          "filter with complex true condition emits empty event",
			configuration: map[string]any{"expression": "$.test == 'value'"},
			inputData:     map[string]any{"test": "value"},
		},
		{
			name:          "filter with complex false condition emits empty event",
			configuration: map[string]any{"expression": "$.test == 'different'"},
			inputData:     map[string]any{"test": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			filter := &Filter{}

			mockExecStateCtx := &MockExecutionStateContext{}

			if tt.configuration["expression"] == "true" || (tt.configuration["expression"] == "$.test == 'value'" && tt.inputData.(map[string]any)["test"] == "value") {
				mockExecStateCtx.On("Pass", map[string][]any{"default": {}}).Return(nil)
			} else {
				mockExecStateCtx.On("Pass", map[string][]any{}).Return(nil)
			}

			ctx := components.ExecutionContext{
				Data:                  tt.inputData,
				Configuration:         tt.configuration,
				ExecutionStateContext: mockExecStateCtx,
			}

			err := filter.Execute(ctx)

			assert.NoError(t, err)

			mockExecStateCtx.AssertExpectations(t)
		})
	}
}

func TestFilter_Execute_InvalidExpression_ShouldReturnError(t *testing.T) {
	filter := &Filter{}

	mockExecStateCtx := &MockExecutionStateContext{}

	ctx := components.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "invalid expression syntax +++"},
		ExecutionStateContext: mockExecStateCtx,
	}

	err := filter.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression compilation failed")
}

func TestFilter_Execute_NonBooleanResult_ShouldReturnError(t *testing.T) {
	filter := &Filter{}

	mockExecStateCtx := &MockExecutionStateContext{}

	ctx := components.ExecutionContext{
		Data:                  map[string]any{"test": "value"},
		Configuration:         map[string]any{"expression": "$.test"},
		ExecutionStateContext: mockExecStateCtx,
	}

	err := filter.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expression must evaluate to boolean")
}
