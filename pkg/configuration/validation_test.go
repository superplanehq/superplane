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
				"startTime":     "09:00",
				"endTime":       "17:00",
				"startDateTime": "2024-12-31T09:00",
				"endDateTime":   "2024-12-31T17:00",
			},
			expectError: false,
		},
		{
			name: "invalid time range - start after end",
			config: map[string]any{
				"startTime":     "17:00",
				"endTime":       "09:00",
				"startDateTime": "2024-12-31T09:00",
				"endDateTime":   "2024-12-31T17:00",
			},
			expectError: true,
			errorMsg:    "start time must be before end time",
		},
		{
			name: "invalid datetime range - start after end",
			config: map[string]any{
				"startTime":     "09:00",
				"endTime":       "17:00",
				"startDateTime": "2024-12-31T17:00",
				"endDateTime":   "2024-12-31T09:00",
			},
			expectError: true,
			errorMsg:    "start date & time must be before end date & time",
		},
		{
			name: "equal times - should fail",
			config: map[string]any{
				"startTime":     "09:00",
				"endTime":       "09:00",
				"startDateTime": "2024-12-31T09:00",
				"endDateTime":   "2024-12-31T17:00",
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

func TestValidateConfiguration_DaysOfWeek(t *testing.T) {
	fields := []Field{
		{
			Name:     "days",
			Type:     FieldTypeDaysOfWeek,
			Required: true,
		},
	}

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
	}{
		{
			name: "valid days list",
			config: map[string]any{
				"days": []any{"monday", "wednesday", "friday"},
			},
			expectError: false,
		},
		{
			name: "empty days list",
			config: map[string]any{
				"days": []any{},
			},
			expectError: true,
		},
		{
			name: "invalid day value",
			config: map[string]any{
				"days": []any{"monday", "funday"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(fields, tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfiguration_TimeRange(t *testing.T) {
	fields := []Field{
		{
			Name:     "timeRange",
			Type:     FieldTypeTimeRange,
			Required: true,
		},
	}

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
	}{
		{
			name: "valid time range",
			config: map[string]any{
				"timeRange": "09:00-17:00",
			},
			expectError: false,
		},
		{
			name: "invalid format",
			config: map[string]any{
				"timeRange": "09:00/17:00",
			},
			expectError: true,
		},
		{
			name: "start after end",
			config: map[string]any{
				"timeRange": "18:00-09:00",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(fields, tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTime_CustomFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		value       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid time with default format 15:04",
			format:      "",
			value:       "18:27",
			expectError: false,
		},
		{
			name:        "valid time with explicit 15:04 format",
			format:      "15:04",
			value:       "18:27",
			expectError: false,
		},
		{
			name:        "invalid time with HH:MM format",
			format:      "HH:MM",
			value:       "18:27",
			expectError: true,
			errorMsg:    "must be a valid time in format HH:MM",
		},
		{
			name:        "valid time with single digit hour",
			format:      "15:04",
			value:       "9:30",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := Field{
				Name: "time",
				Type: FieldTypeTime,
			}

			if tt.format != "" {
				field.TypeOptions = &TypeOptions{
					Time: &TimeTypeOptions{
						Format: tt.format,
					},
				}
			}

			config := map[string]any{
				"time": tt.value,
			}

			err := ValidateConfiguration([]Field{field}, config)
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

func TestValidateList_MaxItems(t *testing.T) {
	tests := []struct {
		name        string
		field       Field
		value       any
		expectError bool
		errorMsg    string
	}{
		{
			name: "list with MaxItems limit - within limit",
			field: Field{
				Name:     "items",
				Type:     FieldTypeList,
				Required: true,
				TypeOptions: &TypeOptions{
					List: &ListTypeOptions{
						MaxItems: ptrInt(3),
						ItemDefinition: &ListItemDefinition{
							Type: FieldTypeString,
						},
					},
				},
			},
			value:       []any{"item1", "item2"},
			expectError: false,
		},
		{
			name: "list with MaxItems limit - at limit",
			field: Field{
				Name:     "items",
				Type:     FieldTypeList,
				Required: true,
				TypeOptions: &TypeOptions{
					List: &ListTypeOptions{
						MaxItems: ptrInt(3),
						ItemDefinition: &ListItemDefinition{
							Type: FieldTypeString,
						},
					},
				},
			},
			value:       []any{"item1", "item2", "item3"},
			expectError: false,
		},
		{
			name: "list with MaxItems limit - exceeds limit",
			field: Field{
				Name:     "items",
				Type:     FieldTypeList,
				Required: true,
				TypeOptions: &TypeOptions{
					List: &ListTypeOptions{
						MaxItems: ptrInt(3),
						ItemDefinition: &ListItemDefinition{
							Type: FieldTypeString,
						},
					},
				},
			},
			value:       []any{"item1", "item2", "item3", "item4"},
			expectError: true,
			errorMsg:    "must contain at most 3 items",
		},
		{
			name: "list without MaxItems limit",
			field: Field{
				Name:     "items",
				Type:     FieldTypeList,
				Required: true,
				TypeOptions: &TypeOptions{
					List: &ListTypeOptions{
						ItemDefinition: &ListItemDefinition{
							Type: FieldTypeString,
						},
					},
				},
			},
			value:       []any{"item1", "item2", "item3", "item4", "item5"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateList(tt.field, tt.value)
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

func ptrInt(v int) *int {
	return &v
}
