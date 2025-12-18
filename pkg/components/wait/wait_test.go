package wait

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

type mockExecutionStateContext struct {
	finished bool
	passed   bool
	failed   bool
}

func (m *mockExecutionStateContext) SetKV(key, value string) error {
	return nil
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
	ctx := core.ActionContext{
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
	ctx := core.ActionContext{
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
	ctx := core.ActionContext{
		Name:                  "unknown",
		ExecutionStateContext: mockState,
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.Error(t, err)
	assert.False(t, mockState.passed)
	assert.False(t, mockState.failed)
}

type mockRequestContext struct {
	scheduledDuration time.Duration
	scheduledAction   string
	scheduledParams   map[string]any
}

func (m *mockRequestContext) ScheduleActionCall(action string, params map[string]any, duration time.Duration) error {
	m.scheduledAction = action
	m.scheduledParams = params
	m.scheduledDuration = duration
	return nil
}

func TestEvaluateIntegerExpression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		data       any
		expected   int
		expectErr  bool
	}{
		{
			name:       "simple integer",
			expression: "5",
			data:       map[string]any{},
			expected:   5,
			expectErr:  false,
		},
		{
			name:       "data field access",
			expression: "$.wait_time",
			data:       map[string]any{"wait_time": 30},
			expected:   30,
			expectErr:  false,
		},
		{
			name:       "arithmetic expression",
			expression: "$.wait_time + 5",
			data:       map[string]any{"wait_time": 25},
			expected:   30,
			expectErr:  false,
		},
		{
			name:       "conditional expression",
			expression: "$.status == \"urgent\" ? 0 : 30",
			data:       map[string]any{"status": "urgent"},
			expected:   0,
			expectErr:  false,
		},
		{
			name:       "conditional expression false",
			expression: "$.status == \"urgent\" ? 0 : 30",
			data:       map[string]any{"status": "normal"},
			expected:   30,
			expectErr:  false,
		},
		{
			name:       "string number",
			expression: "\"42\"",
			data:       map[string]any{},
			expected:   42,
			expectErr:  false,
		},
		{
			name:       "invalid expression",
			expression: "$.nonexistent.field",
			data:       map[string]any{},
			expected:   0,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateIntegerExpression(tt.expression, tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEvaluateDateExpression(t *testing.T) {
	fixedTime := time.Date(2023, 12, 17, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		expression string
		data       any
		expected   time.Time
		expectErr  bool
	}{
		{
			name:       "RFC3339 string",
			expression: "\"2023-12-17T15:30:00Z\"",
			data:       map[string]any{},
			expected:   fixedTime,
			expectErr:  false,
		},
		{
			name:       "data field access",
			expression: "$.run_time",
			data:       map[string]any{"run_time": "2023-12-17T15:30:00Z"},
			expected:   fixedTime,
			expectErr:  false,
		},
		{
			name:       "invalid date format",
			expression: "\"not-a-date\"",
			data:       map[string]any{},
			expected:   time.Time{},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateDateExpression(tt.expression, tt.data)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.expected.Equal(result))
			}
		})
	}
}

func TestCalculateIntervalDuration(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		unit      string
		expected  time.Duration
		expectErr bool
	}{
		{
			name:      "seconds",
			value:     30,
			unit:      "seconds",
			expected:  30 * time.Second,
			expectErr: false,
		},
		{
			name:      "minutes",
			value:     5,
			unit:      "minutes",
			expected:  5 * time.Minute,
			expectErr: false,
		},
		{
			name:      "hours",
			value:     2,
			unit:      "hours",
			expected:  2 * time.Hour,
			expectErr: false,
		},
		{
			name:      "invalid unit",
			value:     10,
			unit:      "days",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "zero value",
			value:     0,
			unit:      "seconds",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "negative value",
			value:     -5,
			unit:      "minutes",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculateIntervalDuration(tt.value, tt.unit)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestWait_Execute_IntervalMode(t *testing.T) {
	w := &Wait{}
	mockRequest := &mockRequestContext{}

	tests := []struct {
		name        string
		config      map[string]any
		data        map[string]any
		expectedDur time.Duration
		expectErr   bool
	}{
		{
			name: "simple interval",
			config: map[string]any{
				"mode":    ModeInterval,
				"waitFor": "30",
				"unit":    "seconds",
			},
			data:        map[string]any{},
			expectedDur: 30 * time.Second,
			expectErr:   false,
		},
		{
			name: "expression interval",
			config: map[string]any{
				"mode":    ModeInterval,
				"waitFor": "$.wait_time + 10",
				"unit":    "minutes",
			},
			data:        map[string]any{"wait_time": 5},
			expectedDur: 15 * time.Minute,
			expectErr:   false,
		},
		{
			name: "conditional expression",
			config: map[string]any{
				"mode":    ModeInterval,
				"waitFor": "$.status == \"urgent\" ? 0 : 60",
				"unit":    "seconds",
			},
			data:        map[string]any{"status": "normal"},
			expectedDur: 60 * time.Second,
			expectErr:   false,
		},
		{
			name: "missing waitFor",
			config: map[string]any{
				"mode": ModeInterval,
				"unit": "seconds",
			},
			data:      map[string]any{},
			expectErr: true,
		},
		{
			name: "missing unit",
			config: map[string]any{
				"mode":    ModeInterval,
				"waitFor": "30",
			},
			data:      map[string]any{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.ExecutionContext{
				Configuration:  tt.config,
				Data:           tt.data,
				RequestContext: mockRequest,
			}

			err := w.Execute(ctx)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "timeReached", mockRequest.scheduledAction)
				assert.Equal(t, tt.expectedDur, mockRequest.scheduledDuration)
			}
		})
	}
}

func TestWait_Execute_CountdownMode(t *testing.T) {
	w := &Wait{}
	mockRequest := &mockRequestContext{}

	futureTime := time.Now().Add(1 * time.Hour)
	futureTimeStr := futureTime.Format(time.RFC3339)

	tests := []struct {
		name       string
		config     map[string]any
		data       map[string]any
		expectErr  bool
		expectPast bool
	}{
		{
			name: "future time",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "\"" + futureTimeStr + "\"",
			},
			data:       map[string]any{},
			expectErr:  false,
			expectPast: false,
		},
		{
			name: "expression countdown",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "$.run_time",
			},
			data:       map[string]any{"run_time": futureTimeStr},
			expectErr:  false,
			expectPast: false,
		},
		{
			name: "past time",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "\"2020-01-01T00:00:00Z\"",
			},
			data:       map[string]any{},
			expectErr:  true,
			expectPast: true,
		},
		{
			name: "missing waitUntil",
			config: map[string]any{
				"mode": ModeCountdown,
			},
			data:      map[string]any{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.ExecutionContext{
				Configuration:  tt.config,
				Data:           tt.data,
				RequestContext: mockRequest,
			}

			err := w.Execute(ctx)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.expectPast {
					assert.Contains(t, err.Error(), "in the past")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "timeReached", mockRequest.scheduledAction)

				assert.True(t, mockRequest.scheduledDuration > 59*time.Minute)
				assert.True(t, mockRequest.scheduledDuration < 61*time.Minute)
			}
		})
	}
}

func TestWait_Execute_InvalidConfiguration(t *testing.T) {
	w := &Wait{}
	mockRequest := &mockRequestContext{}

	tests := []struct {
		name   string
		config map[string]any
		errMsg string
	}{
		{
			name:   "empty configuration",
			config: map[string]any{},
			errMsg: "invalid mode",
		},
		{
			name: "invalid mode",
			config: map[string]any{
				"mode": "invalid",
			},
			errMsg: "invalid mode",
		},
		{
			name: "missing waitFor in interval mode",
			config: map[string]any{
				"mode": ModeInterval,
				"unit": "seconds",
			},
			errMsg: "waitFor and unit are required for interval mode",
		},
		{
			name: "missing waitUntil in countdown mode",
			config: map[string]any{
				"mode": ModeCountdown,
			},
			errMsg: "waitUntil is required for countdown mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.ExecutionContext{
				Configuration:  tt.config,
				RequestContext: mockRequest,
			}

			err := w.Execute(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
