package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__ValidateConfiguration__RequiredConditions(t *testing.T) {
	fields := []Field{
		{
			Name:     "filterType",
			Type:     FieldTypeSelect,
			Required: true,
			TypeOptions: &TypeOptions{
				Select: &SelectTypeOptions{
					Options: []FieldOption{
						{Label: "none", Value: "none"},
						{Label: "range", Value: "range"},
					},
				},
			},
		},
		{
			Name: "startTime",
			Type: FieldTypeTime,
			RequiredConditions: []RequiredCondition{
				{
					Field:  "filterType",
					Values: []string{"range"},
				},
			},
		},
		{
			Name: "endTime",
			Type: FieldTypeTime,
			RequiredConditions: []RequiredCondition{
				{
					Field:  "filterType",
					Values: []string{"range"},
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
			name: "nothing is required for none filter",
			config: map[string]any{
				"filterType": "none",
			},
			expectError: false,
		},
		{
			name: "startTime required for range filter",
			config: map[string]any{
				"filterType": "range",
				// startTime missing - should fail
			},
			expectError: true,
			errorMsg:    "field 'startTime' is required",
		},
		{
			name: "endTime required for range filter",
			config: map[string]any{
				"filterType": "range",
				"startTime":  "09:00",
				// endTime missing - should fail
			},
			expectError: true,
			errorMsg:    "field 'endTime' is required",
		},
		{
			name: "startTime and endTime provided for range mode",
			config: map[string]any{
				"filterType": "range",
				"startTime":  "09:00",
				"endTime":    "17:00",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(fields, tt.config)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test__ValidateConfiguration__DaysOfWeek(t *testing.T) {
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

func Test__ValidateConfiguration__TimeRange(t *testing.T) {
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

func Test__ValidateConfiguration__CustomTimeFormat(t *testing.T) {
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

func Test__ValidateList__MaxItems(t *testing.T) {
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
