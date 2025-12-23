package wait

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestWait_HandleAction_PushThrough(t *testing.T) {
	w := &Wait{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionContext{
		Name:                  "pushThrough",
		ExecutionStateContext: stateCtx,
		MetadataContext:       &contexts.MetadataContext{},
		AuthContext:           &contexts.AuthContext{},
		Configuration:         nil,
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.IsFinished())
}

func TestWait_HandleAction_TimeReached(t *testing.T) {
	w := &Wait{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionContext{
		Name:                  "timeReached",
		ExecutionStateContext: stateCtx,
		MetadataContext:       &contexts.MetadataContext{},
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)
}

func TestWait_HandleAction_Unknown(t *testing.T) {
	w := &Wait{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionContext{
		Name:                  "unknown",
		ExecutionStateContext: stateCtx,
		MetadataContext:       &contexts.MetadataContext{},
		Parameters:            map[string]any{},
	}

	err := w.HandleAction(ctx)
	assert.Error(t, err)
	assert.False(t, stateCtx.Passed)
	assert.False(t, stateCtx.Finished)
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
			requestCtx := &contexts.RequestContext{}
			ctx := core.ExecutionContext{
				Configuration:         tt.config,
				Data:                  tt.data,
				RequestContext:        requestCtx,
				MetadataContext:       &contexts.MetadataContext{},
				ExecutionStateContext: &contexts.ExecutionStateContext{},
			}

			err := w.Execute(ctx)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "timeReached", requestCtx.Action)
				assert.Equal(t, tt.expectedDur, requestCtx.Duration)
			}
		})
	}
}

func TestWait_Execute_CountdownMode(t *testing.T) {
	w := &Wait{}

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
			requestCtx := &contexts.RequestContext{}
			ctx := core.ExecutionContext{
				Configuration:         tt.config,
				Data:                  tt.data,
				RequestContext:        requestCtx,
				MetadataContext:       &contexts.MetadataContext{},
				ExecutionStateContext: &contexts.ExecutionStateContext{},
			}

			err := w.Execute(ctx)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.expectPast {
					assert.Contains(t, err.Error(), "in the past")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "timeReached", requestCtx.Action)

				assert.True(t, requestCtx.Duration > 59*time.Minute)
				assert.True(t, requestCtx.Duration < 61*time.Minute)
			}
		})
	}
}

func TestWait_Execute_InvalidConfiguration(t *testing.T) {
	w := &Wait{}

	tests := []struct {
		name   string
		config map[string]any
		errMsg string
	}{
		{
			name:   "empty configuration",
			config: map[string]any{},
			errMsg: "Invalid mode:",
		},
		{
			name: "invalid mode",
			config: map[string]any{
				"mode": "invalid",
			},
			errMsg: "Invalid mode:",
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
				Configuration:         tt.config,
				RequestContext:        &contexts.RequestContext{},
				MetadataContext:       &contexts.MetadataContext{},
				ExecutionStateContext: &contexts.ExecutionStateContext{},
			}

			err := w.Execute(ctx)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestWait_HandleTimeReached_CompletionOutput(t *testing.T) {
	w := &Wait{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionContext{
		Name:                  "timeReached",
		ExecutionStateContext: stateCtx,
		MetadataContext: &contexts.MetadataContext{
			Metadata: ExecutionMetadata{StartTime: "2025-12-10T09:02:43.651Z"},
		},
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)

	// Check completion output structure
	require.Len(t, stateCtx.Outputs, 1)
	output := stateCtx.Outputs[0]

	assert.Equal(t, output.Channel, core.DefaultOutputChannel.Name)
	require.Len(t, output.Payloads, 1)

	outputData := output.Payloads[0].Data.(map[string]any)
	assert.Equal(t, "2025-12-10T09:02:43.651Z", outputData["started_at"])
	assert.Equal(t, "completed", outputData["result"])
	assert.Equal(t, "timeout", outputData["reason"])
	assert.Nil(t, outputData["actor"])
	assert.Contains(t, outputData, "finished_at")
}

func TestWait_HandlePushThrough_CompletionOutput(t *testing.T) {
	w := &Wait{}

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ActionContext{
		Name:                  "pushThrough",
		ExecutionStateContext: stateCtx,
		AuthContext: &contexts.AuthContext{
			User: &core.User{
				Email: "alex@company.com",
				Name:  "Aleksandar Mitrović",
			},
		},
		MetadataContext: &contexts.MetadataContext{
			Metadata: ExecutionMetadata{StartTime: "2025-12-10T09:02:43.651Z"},
		},
	}

	err := w.HandleAction(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)

	// Check completion output structure
	require.Len(t, stateCtx.Outputs, 1)
	output := stateCtx.Outputs[0]

	assert.Equal(t, output.Channel, core.DefaultOutputChannel.Name)
	require.Len(t, output.Payloads, 1)
	outputData := output.Payloads[0].Data.(map[string]any)
	assert.Equal(t, "2025-12-10T09:02:43.651Z", outputData["started_at"])
	assert.Equal(t, "completed", outputData["result"])
	assert.Equal(t, "manual_override", outputData["reason"])
	assert.Contains(t, outputData, "finished_at")

	// Check actor information
	assert.NotNil(t, outputData["actor"])
	actor := outputData["actor"].(map[string]any)
	assert.Equal(t, "alex@company.com", actor["email"])
	assert.Equal(t, "Aleksandar Mitrović", actor["display_name"])
}

func TestWait_Execute_CountdownMode_ResolvedValues(t *testing.T) {
	w := &Wait{}

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
			errMsg:    "Failed to parse waitUntil value",
		},
		{
			name: "empty string",
			config: map[string]any{
				"mode":      ModeCountdown,
				"waitUntil": "",
			},
			expectErr: true,
			errMsg:    "Failed to parse waitUntil value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := core.ExecutionContext{
				Configuration:         tt.config,
				RequestContext:        &contexts.RequestContext{},
				MetadataContext:       &contexts.MetadataContext{},
				ExecutionStateContext: &contexts.ExecutionStateContext{},
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

	stateCtx := &contexts.ExecutionStateContext{}
	ctx := core.ExecutionContext{
		ExecutionStateContext: stateCtx,
		MetadataContext: &contexts.MetadataContext{
			Metadata: ExecutionMetadata{StartTime: "2025-12-10T09:02:43.651Z"},
		},
		AuthContext: &contexts.AuthContext{
			User: &core.User{
				Email: "alex@company.com",
				Name:  "Aleksandar Mitrović",
			},
		},
	}

	err := w.Cancel(ctx)
	assert.NoError(t, err)
	assert.True(t, stateCtx.Passed)
	assert.True(t, stateCtx.Finished)

	// Check completion output structure
	require.Len(t, stateCtx.Outputs, 1)
	assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Outputs[0].Channel)
	payloads := stateCtx.Outputs[0].Payloads
	assert.Len(t, payloads, 1)
	output := payloads[0].Data.(map[string]any)
	assert.Equal(t, "2025-12-10T09:02:43.651Z", output["started_at"])
	assert.Equal(t, "cancelled", output["result"])
	assert.Equal(t, "user_cancel", output["reason"])
	assert.Contains(t, output, "finished_at")

	// Check actor information
	assert.NotNil(t, output["actor"])
	actor := output["actor"].(map[string]any)
	assert.Equal(t, "alex@company.com", actor["email"])
	assert.Equal(t, "Aleksandar Mitrović", actor["display_name"])
}
