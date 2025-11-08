package noop

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

			mockExecStateCtx := &MockExecutionStateContext{}

			mockExecStateCtx.On("Pass", map[string][]any{}).Return(nil)

			ctx := components.ExecutionContext{
				Data:                  tt.inputData,
				ExecutionStateContext: mockExecStateCtx,
			}

			err := noop.Execute(ctx)

			assert.NoError(t, err)

			mockExecStateCtx.AssertExpectations(t)
		})
	}
}

func TestNoop_Execute_AlwaysEmitsEmpty(t *testing.T) {
	t.Run("noop should never pass through original data", func(t *testing.T) {
		noop := &NoOp{}
		mockExecStateCtx := &MockExecutionStateContext{}

		originalData := map[string]any{
			"important": "data",
			"that":      "should not be passed through",
		}

		mockExecStateCtx.AssertNotCalled(t, "Pass", map[string][]any{
			components.DefaultOutputChannel.Name: {originalData},
		})

		mockExecStateCtx.On("Pass", map[string][]any{}).Return(nil)

		ctx := components.ExecutionContext{
			Data:                  originalData,
			ExecutionStateContext: mockExecStateCtx,
		}

		err := noop.Execute(ctx)
		assert.NoError(t, err)
		mockExecStateCtx.AssertExpectations(t)
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
	assert.Equal(t, "check", noop.Icon())
	assert.Equal(t, "blue", noop.Color())
}
