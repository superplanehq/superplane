package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfiguration_RequiredConditions(t *testing.T) {
	fields := []Field{
		{
			Name:     "mode",
			Type:     FieldTypeSelect,
			Required: true,
		},
		{
			Name:     "startTime",
			Type:     FieldTypeTime,
			Required: false,
			RequiredConditions: []RequiredCondition{
				{
					Field:  "mode",
					Values: []string{"include_range", "exclude_range"},
				},
			},
		},
		{
			Name:     "startDateTime",
			Type:     FieldTypeDateTime,
			Required: false,
			RequiredConditions: []RequiredCondition{
				{
					Field:  "mode",
					Values: []string{"include_specific", "exclude_specific"},
				},
			},
		},
	}

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "startTime required for range mode",
			config: map[string]any{
				"mode": "include_range",
				// startTime missing - should fail
			},
			expectError: true,
			errorMsg:    "field 'startTime' is required",
		},
		{
			name: "startTime provided for range mode",
			config: map[string]any{
				"mode":      "include_range",
				"startTime": "09:00",
			},
			expectError: false,
		},
		{
			name: "startDateTime required for specific mode",
			config: map[string]any{
				"mode": "include_specific",
				// startDateTime missing - should fail
			},
			expectError: true,
			errorMsg:    "field 'startDateTime' is required",
		},
		{
			name: "startDateTime provided for specific mode",
			config: map[string]any{
				"mode":          "include_specific",
				"startDateTime": "2024-12-31T00:00",
			},
			expectError: false,
		},
		{
			name: "startTime not required for specific mode",
			config: map[string]any{
				"mode":          "include_specific",
				"startDateTime": "2024-12-31T00:00",
				// startTime not provided - should pass
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(fields, tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfiguration_ValidationRules(t *testing.T) {
	fields := []Field{
		{
			Name:     "startTime",
			Type:     FieldTypeTime,
			Required: true,
			ValidationRules: []ValidationRule{
				{
					Type:        ValidationRuleLessThan,
					CompareWith: "endTime",
					Message:     "start time must be before end time",
				},
			},
		},
		{
			Name:     "endTime",
			Type:     FieldTypeTime,
			Required: true,
		},
		{
			Name:     "startDateTime",
			Type:     FieldTypeDateTime,
			Required: true,
			ValidationRules: []ValidationRule{
				{
					Type:        ValidationRuleLessThan,
					CompareWith: "endDateTime",
					Message:     "start date & time must be before end date & time",
				},
			},
		},
		{
			Name:     "endDateTime",
			Type:     FieldTypeDateTime,
			Required: true,
		},
	}

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid time range",
			config: map[string]any{
				"startTime":      "09:00",
				"endTime":        "17:00",
				"startDateTime":  "2024-12-31T09:00",
				"endDateTime":    "2024-12-31T17:00",
			},
			expectError: false,
		},
		{
			name: "invalid time range - start after end",
			config: map[string]any{
				"startTime":      "17:00",
				"endTime":        "09:00",
				"startDateTime":  "2024-12-31T09:00",
				"endDateTime":    "2024-12-31T17:00",
			},
			expectError: true,
			errorMsg:    "start time must be before end time",
		},
		{
			name: "invalid datetime range - start after end",
			config: map[string]any{
				"startTime":      "09:00",
				"endTime":        "17:00",
				"startDateTime":  "2024-12-31T17:00",
				"endDateTime":    "2024-12-31T09:00",
			},
			expectError: true,
			errorMsg:    "start date & time must be before end date & time",
		},
		{
			name: "equal times - should fail",
			config: map[string]any{
				"startTime":      "09:00",
				"endTime":        "09:00",
				"startDateTime":  "2024-12-31T09:00",
				"endDateTime":    "2024-12-31T17:00",
			},
			expectError: true,
			errorMsg:    "start time must be before end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(fields, tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDayInYear(t *testing.T) {
	field := Field{
		Name: "testDay",
		Type: FieldTypeDayInYear,
	}

	tests := []struct {
		name        string
		value       any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid Christmas",
			value:       "12/25",
			expectError: false,
		},
		{
			name:        "valid New Year",
			value:       "01/01",
			expectError: false,
		},
		{
			name:        "valid leap day",
			value:       "02/29",
			expectError: false,
		},
		{
			name:        "valid July 4th",
			value:       "07/04",
			expectError: false,
		},
		{
			name:        "single digit month and day",
			value:       "1/1",
			expectError: false,
		},
		{
			name:        "not a string",
			value:       123,
			expectError: true,
			errorMsg:    "must be a string",
		},
		{
			name:        "invalid format",
			value:       "invalid",
			expectError: true,
			errorMsg:    "must be a valid day",
		},
		{
			name:        "invalid month",
			value:       "13/01",
			expectError: true,
			errorMsg:    "invalid day values",
		},
		{
			name:        "invalid day",
			value:       "01/32",
			expectError: true,
			errorMsg:    "invalid day values",
		},
		{
			name:        "zero month",
			value:       "00/15",
			expectError: true,
			errorMsg:    "invalid day values",
		},
		{
			name:        "zero day",
			value:       "06/00",
			expectError: true,
			errorMsg:    "invalid day values",
		},
		{
			name:        "invalid day for February",
			value:       "02/30",
			expectError: true,
			errorMsg:    "invalid day '30' for month '2'",
		},
		{
			name:        "invalid day for April",
			value:       "04/31",
			expectError: true,
			errorMsg:    "invalid day '31' for month '4'",
		},
		{
			name:        "extra parts",
			value:       "12/25/2024",
			expectError: true,
			errorMsg:    "must be a valid day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDayInYear(field, tt.value)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDayInYearComparison(t *testing.T) {
	fields := []Field{
		{
			Name:     "startDayInYear",
			Type:     FieldTypeDayInYear,
			Required: true,
			ValidationRules: []ValidationRule{
				{
					Type:        ValidationRuleLessThan,
					CompareWith: "endDayInYear",
					Message:     "start day must be before end day",
				},
			},
		},
		{
			Name:     "endDayInYear",
			Type:     FieldTypeDayInYear,
			Required: true,
		},
	}

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid day range",
			config: map[string]any{
				"startDayInYear": "12/25",
				"endDayInYear":   "12/31",
			},
			expectError: false,
		},
		{
			name: "invalid day range - start after end",
			config: map[string]any{
				"startDayInYear": "12/31",
				"endDayInYear":   "12/25",
			},
			expectError: true,
			errorMsg:    "start day must be before end day",
		},
		{
			name: "cross-year range - valid",
			config: map[string]any{
				"startDayInYear": "12/25",
				"endDayInYear":   "01/05",
			},
			expectError: false, // Cross-year ranges are allowed
		},
		{
			name: "same day - valid",
			config: map[string]any{
				"startDayInYear": "07/04",
				"endDayInYear":   "07/04",
			},
			expectError: true, // Same day should fail for less_than comparison
			errorMsg:    "start day must be before end day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(fields, tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}