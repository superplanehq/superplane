package wait

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/components"
)

// mockExecutionStateContext implements components.ExecutionStateContext for tests
type mockExecutionStateContext struct {
	finished bool
	passed   bool
	failed   bool
}

func (m *mockExecutionStateContext) IsFinished() bool { return m.finished }
func (m *mockExecutionStateContext) Pass(outputs map[string][]any) error {
	m.passed = true
	m.finished = true
	return nil
}
func (m *mockExecutionStateContext) Fail(reason, message string) error {
	m.failed = true
	m.finished = true
	return nil
}

func TestWait_HandleAction_PushThrough(t *testing.T) {
	w := &Wait{}

	mockState := &mockExecutionStateContext{}
	ctx := components.ActionContext{
		Name:                  "pushThrough",
		ExecutionStateContext: mockState,
		MetadataContext:       nil,
		Configuration:         nil,
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, mockState.passed)
	assert.True(t, mockState.finished)
}

func TestWait_HandleAction_TimeReached(t *testing.T) {
	w := &Wait{}

	mockState := &mockExecutionStateContext{}
	ctx := components.ActionContext{
		Name:                  "timeReached",
		ExecutionStateContext: mockState,
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, mockState.passed)
	assert.True(t, mockState.finished)
}

func TestWait_HandleAction_Unknown(t *testing.T) {
	w := &Wait{}

	mockState := &mockExecutionStateContext{}
	ctx := components.ActionContext{
		Name:                  "unknown",
		ExecutionStateContext: mockState,
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.Error(t, err)
	assert.False(t, mockState.passed)
	assert.False(t, mockState.failed)
}
