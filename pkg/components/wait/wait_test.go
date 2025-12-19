package wait

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

type mockExecutionStateContext struct {
	finished   bool
	passed     bool
	failed     bool
	passedData map[string][]any
}

func (m *mockExecutionStateContext) SetKV(key, value string) error {
	return nil
}

func (m *mockExecutionStateContext) IsFinished() bool { return m.finished }
func (m *mockExecutionStateContext) Pass(outputs map[string][]any) error {
	m.passed = true
	m.finished = true
	m.passedData = outputs
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
	mockMetadata := &mockMetadataContext{}
	mockAuth := &mockAuthContext{}

	ctx := core.ActionContext{
		Name:                  "pushThrough",
		ExecutionStateContext: mockState,
		MetadataContext:       mockMetadata,
		AuthContext:           mockAuth,
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
	mockMetadata := &mockMetadataContext{}

	ctx := core.ActionContext{
		Name:                  "timeReached",
		ExecutionStateContext: mockState,
		MetadataContext:       mockMetadata,
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
	mockMetadata := &mockMetadataContext{}

	ctx := core.ActionContext{
		Name:                  "unknown",
		ExecutionStateContext: mockState,
		MetadataContext:       mockMetadata,
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

// mockMetadataContext implements core.MetadataContext for tests
type mockMetadataContext struct {
	data any
}

func (m *mockMetadataContext) Get() any {
	return m.data
}

func (m *mockMetadataContext) Set(data any) {
	m.data = data
}

// mockAuthContext implements core.AuthContext for tests
type mockAuthContext struct {
	user *core.User
}

func (m *mockAuthContext) AuthenticatedUser() *core.User {
	return m.user
}

func (m *mockAuthContext) GetUser(id uuid.UUID) (*core.User, error) {
	return nil, nil
}

func (m *mockAuthContext) HasRole(role string) (bool, error) {
	return false, nil
}

func (m *mockAuthContext) InGroup(group string) (bool, error) {
	return false, nil
}

func TestParseIntegerValue(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		expected  int
		expectErr bool
	}{
		{
			name:      "int value",
			value:     5,
			expected:  5,
			expectErr: false,
		},
		{
			name:      "int64 value",
			value:     int64(30),
			expected:  30,
			expectErr: false,
		},
		{
			name:      "float64 value",
			value:     float64(25),
			expected:  25,
			expectErr: false,
		},
		{
			name:      "string number",
			value:     "42",
			expected:  42,
			expectErr: false,
		},
		{
			name:      "invalid string",
			value:     "not a number",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "invalid type",
			value:     []string{"invalid"},
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIntegerValue(tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseDateValue(t *testing.T) {
	fixedTime := time.Date(2023, 12, 17, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		value     any
		expected  time.Time
		expectErr bool
	}{
		{
			name:      "RFC3339 string",
			value:     "2023-12-17T15:30:00Z",
			expected:  fixedTime,
			expectErr: false,
		},
		{
			name:      "time.Time value",
			value:     fixedTime,
			expected:  fixedTime,
			expectErr: false,
		},
		{
			name:      "alternative format string",
			value:     "2023-12-17 15:30:00",
			expected:  fixedTime,
			expectErr: false,
		},
		{
			name:      "invalid date format",
			value:     "not-a-date",
			expected:  time.Time{},
			expectErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			expected:  time.Time{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDateValue(tt.value)
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
			name: "resolved integer value",
			config: map[string]any{
				"mode":    ModeInterval,
				"waitFor": 15,
				"unit":    "minutes",
			},
			data:        map[string]any{},
			expectedDur: 15 * time.Minute,
			expectErr:   false,
		},
		{
			name: "resolved string integer",
			config: map[string]any{
				"mode":    ModeInterval,
				"waitFor": "60",
				"unit":    "seconds",
			},
			data:        map[string]any{},
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
			mockMetadata := &mockMetadataContext{}
			ctx := core.ExecutionContext{
				Configuration:   tt.config,
				Data:            tt.data,
				RequestContext:  mockRequest,
				MetadataContext: mockMetadata,
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
				"waitUntil": futureTimeStr,
			},
			data:       map[string]any{},
			expectErr:  false,
			expectPast: false,
		},
		{
			name: "resolved future time",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": futureTimeStr,
			},
			data:       map[string]any{},
			expectErr:  false,
			expectPast: false,
		},
		{
			name: "past time",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "2020-01-01T00:00:00Z",
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
			mockMetadata := &mockMetadataContext{}
			ctx := core.ExecutionContext{
				Configuration:   tt.config,
				Data:            tt.data,
				RequestContext:  mockRequest,
				MetadataContext: mockMetadata,
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
			mockMetadata := &mockMetadataContext{}
			ctx := core.ExecutionContext{
				Configuration:   tt.config,
				RequestContext:  mockRequest,
				MetadataContext: mockMetadata,
			}

			err := w.Execute(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestWait_HandleTimeReached_CompletionOutput(t *testing.T) {
	w := &Wait{}

	mockState := &mockExecutionStateContext{}
	mockMetadata := &mockMetadataContext{}
	// Set up metadata with start time
	mockMetadata.Set(ExecutionMetadata{StartTime: "2025-12-10T09:02:43.651Z"})

	ctx := core.ActionContext{
		Name:                  "timeReached",
		ExecutionStateContext: mockState,
		MetadataContext:       mockMetadata,
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, mockState.passed)
	assert.True(t, mockState.finished)

	// Check completion output structure
	assert.Contains(t, mockState.passedData, core.DefaultOutputChannel.Name)
	outputs := mockState.passedData[core.DefaultOutputChannel.Name]
	assert.Len(t, outputs, 1)

	output := outputs[0].(map[string]any)
	assert.Equal(t, "2025-12-10T09:02:43.651Z", output["timestamp_started"])
	assert.Equal(t, "completed", output["result"])
	assert.Equal(t, "timeout", output["reason"])
	assert.Nil(t, output["actor"])
	assert.Contains(t, output, "timestamp_finished")
}

func TestWait_HandlePushThrough_CompletionOutput(t *testing.T) {
	w := &Wait{}

	mockState := &mockExecutionStateContext{}
	mockMetadata := &mockMetadataContext{}
	mockAuth := &mockAuthContext{
		user: &core.User{
			Email: "alex@company.com",
			Name:  "Aleksandar Mitrović",
		},
	}
	// Set up metadata with start time
	mockMetadata.Set(ExecutionMetadata{StartTime: "2025-12-10T09:02:43.651Z"})

	ctx := core.ActionContext{
		Name:                  "pushThrough",
		ExecutionStateContext: mockState,
		MetadataContext:       mockMetadata,
		AuthContext:           mockAuth,
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, mockState.passed)
	assert.True(t, mockState.finished)

	// Check completion output structure
	assert.Contains(t, mockState.passedData, core.DefaultOutputChannel.Name)
	outputs := mockState.passedData[core.DefaultOutputChannel.Name]
	assert.Len(t, outputs, 1)

	output := outputs[0].(map[string]any)
	assert.Equal(t, "2025-12-10T09:02:43.651Z", output["timestamp_started"])
	assert.Equal(t, "completed", output["result"])
	assert.Equal(t, "manual_override", output["reason"])
	assert.Contains(t, output, "timestamp_finished")

	// Check actor information
	assert.NotNil(t, output["actor"])
	actor := output["actor"].(map[string]any)
	assert.Equal(t, "alex@company.com", actor["email"])
	assert.Equal(t, "Aleksandar Mitrović", actor["display_name"])
}

func TestWait_Execute_CountdownMode_ResolvedValues(t *testing.T) {
	w := &Wait{}
	mockRequest := &mockRequestContext{}

	futureTime := time.Now().Add(1 * time.Hour)
	futureTimeStr := futureTime.Format(time.RFC3339)

	tests := []struct {
		name      string
		config    map[string]any
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid ISO 8601 string",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": futureTimeStr,
			},
			expectErr: false,
		},
		{
			name: "alternative date format",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "2025-12-20 15:30:00",
			},
			expectErr: false,
		},
		{
			name: "invalid date format",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "not-a-date",
			},
			expectErr: true,
			errMsg:    "failed to parse waitUntil value",
		},
		{
			name: "empty string",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "",
			},
			expectErr: true,
			errMsg:    "failed to parse waitUntil value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMetadata := &mockMetadataContext{}
			ctx := core.ExecutionContext{
				Configuration:   tt.config,
				RequestContext:  mockRequest,
				MetadataContext: mockMetadata,
			}

			err := w.Execute(ctx)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				// For valid values, we might still get an error if the resulting date is in the past
				// That's expected behavior, not a parsing error
				if err != nil {
					// If there's an error, it should be about the date being in the past, not parsing
					assert.Contains(t, err.Error(), "in the past")
				}
			}
		})
	}
}

func TestWait_Cancel_CompletionOutput(t *testing.T) {
	w := &Wait{}

	mockState := &mockExecutionStateContext{}
	mockMetadata := &mockMetadataContext{}
	mockAuth := &mockAuthContext{
		user: &core.User{
			Email: "alex@company.com",
			Name:  "Aleksandar Mitrović",
		},
	}
	// Set up metadata with start time
	mockMetadata.Set(ExecutionMetadata{StartTime: "2025-12-10T09:02:43.651Z"})

	ctx := core.ExecutionContext{
		ExecutionStateContext: mockState,
		MetadataContext:       mockMetadata,
		AuthContext:           mockAuth,
	}

	err := w.Cancel(ctx)
	assert.NoError(t, err)
	assert.True(t, mockState.passed)
	assert.True(t, mockState.finished)

	// Check completion output structure
	assert.Contains(t, mockState.passedData, core.DefaultOutputChannel.Name)
	outputs := mockState.passedData[core.DefaultOutputChannel.Name]
	assert.Len(t, outputs, 1)

	output := outputs[0].(map[string]any)
	assert.Equal(t, "2025-12-10T09:02:43.651Z", output["timestamp_started"])
	assert.Equal(t, "cancelled", output["result"])
	assert.Equal(t, "user_cancel", output["reason"])
	assert.Contains(t, output, "timestamp_finished")

	// Check actor information
	assert.NotNil(t, output["actor"])
	actor := output["actor"].(map[string]any)
	assert.Equal(t, "alex@company.com", actor["email"])
	assert.Equal(t, "Aleksandar Mitrović", actor["display_name"])
}
