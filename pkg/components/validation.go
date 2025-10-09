package components

import (
	"fmt"
	"net/url"
	"strings"

	"slices"
)

// ValidateConfiguration validates a configuration map against the field definitions
func ValidateConfiguration(fields []ConfigurationField, config map[string]any) error {
	for _, field := range fields {
		value, exists := config[field.Name]

		// Check required fields
		if field.Required && (!exists || value == nil) {
			return fmt.Errorf("field '%s' is required", field.Name)
		}

		// Skip validation if field is not present and not required
		if !exists || value == nil {
			continue
		}

		// Validate based on type
		if err := validateFieldValue(field, value); err != nil {
			return fmt.Errorf("field '%s': %w", field.Name, err)
		}
	}

	return nil
}

func validateFieldValue(field ConfigurationField, value any) error {
	switch field.Type {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeNumber:
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case int:
			num = float64(v)
		case int32:
			num = float64(v)
		case int64:
			num = float64(v)
		default:
			return fmt.Errorf("must be a number")
		}

		if field.Min != nil && num < float64(*field.Min) {
			return fmt.Errorf("must be at least %d", *field.Min)
		}
		if field.Max != nil && num > float64(*field.Max) {
			return fmt.Errorf("must be at most %d", *field.Max)
		}

	case FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}

	case FieldTypeURL:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if _, err := url.ParseRequestURI(str); err != nil {
			return fmt.Errorf("must be a valid URL")
		}

	case FieldTypeDate:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a date string")
		}
		// Additional date format validation could be added here

	case FieldTypeSelect:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if len(field.Options) > 0 {
			valid := slices.ContainsFunc(field.Options, func(opt FieldOption) bool {
				return opt.Value == str
			})
			if !valid {
				validValues := make([]string, len(field.Options))
				for i, opt := range field.Options {
					validValues[i] = opt.Value
				}
				return fmt.Errorf("must be one of: %s", strings.Join(validValues, ", "))
			}
		}

	case FieldTypeMultiSelect:
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("must be an array")
		}
		if len(field.Options) > 0 {
			for _, item := range arr {
				str, ok := item.(string)
				if !ok {
					return fmt.Errorf("all items must be strings")
				}
				valid := slices.ContainsFunc(field.Options, func(opt FieldOption) bool {
					return opt.Value == str
				})
				if !valid {
					validValues := make([]string, len(field.Options))
					for i, opt := range field.Options {
						validValues[i] = opt.Value
					}
					return fmt.Errorf("value '%s' must be one of: %s", str, strings.Join(validValues, ", "))
				}
			}
		}

	case FieldTypeList:
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("must be an array")
		}
		if field.ListItem != nil {
			for i, item := range arr {
				if field.ListItem.Type == FieldTypeObject && len(field.ListItem.Schema) > 0 {
					itemMap, ok := item.(map[string]any)
					if !ok {
						return fmt.Errorf("item at index %d must be an object", i)
					}
					if err := ValidateConfiguration(field.ListItem.Schema, itemMap); err != nil {
						return fmt.Errorf("item at index %d: %w", i, err)
					}
				}
			}
		}

	case FieldTypeObject:
		objMap, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("must be an object")
		}
		if len(field.Schema) > 0 {
			if err := ValidateConfiguration(field.Schema, objMap); err != nil {
				return err
			}
		}
	}

	return nil
}
