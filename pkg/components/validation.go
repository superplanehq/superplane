package components

import (
	"fmt"
	"strings"

	"slices"
)

func ValidateConfiguration(fields []ConfigurationField, config map[string]any) error {
	for _, field := range fields {
		value, exists := config[field.Name]
		if field.Required && (!exists || value == nil) {
			return fmt.Errorf("field '%s' is required", field.Name)
		}

		if !exists || value == nil {
			continue
		}

		err := validateFieldValue(field, value)
		if err != nil {
			return fmt.Errorf("field '%s': %w", field.Name, err)
		}
	}

	return nil
}

func validateNumber(field ConfigurationField, value any) error {
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

	if field.TypeOptions == nil || field.TypeOptions.Number == nil {
		return nil
	}

	options := field.TypeOptions.Number
	if options.Min != nil && num < float64(*options.Min) {
		return fmt.Errorf("must be at least %d", *options.Min)
	}

	if options.Max != nil && num > float64(*options.Max) {
		return fmt.Errorf("must be at most %d", *options.Max)
	}

	return nil
}

func validateSelect(field ConfigurationField, value any) error {
	selected, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if field.TypeOptions == nil || field.TypeOptions.Select == nil {
		return nil
	}

	options := field.TypeOptions.Select
	if len(options.Options) == 0 {
		return nil
	}

	valid := slices.ContainsFunc(options.Options, func(opt FieldOption) bool {
		return opt.Value == selected
	})

	if !valid {
		validValues := make([]string, len(options.Options))
		for i, opt := range options.Options {
			validValues[i] = opt.Value
		}

		return fmt.Errorf("must be one of: %s", strings.Join(validValues, ", "))
	}

	return nil
}

func validateMultiSelect(field ConfigurationField, value any) error {
	selectedValues, ok := value.([]any)
	if !ok {
		return fmt.Errorf("must be a list of values")
	}

	if field.TypeOptions == nil || field.TypeOptions.MultiSelect == nil {
		return nil
	}

	typeOptions := field.TypeOptions.MultiSelect
	if len(typeOptions.Options) == 0 {
		return nil
	}

	for _, selectedValue := range selectedValues {
		v, ok := selectedValue.(string)
		if !ok {
			return fmt.Errorf("all items must be strings")
		}

		valid := slices.ContainsFunc(typeOptions.Options, func(opt FieldOption) bool {
			return opt.Value == v
		})

		if valid {
			continue
		}

		validValues := make([]string, len(typeOptions.Options))
		for i, opt := range typeOptions.Options {
			validValues[i] = opt.Value
		}

		return fmt.Errorf("value '%s' must be one of: %s", v, strings.Join(validValues, ", "))
	}

	return nil
}

func validateObject(field ConfigurationField, value any) error {
	if field.TypeOptions != nil && field.TypeOptions.Object != nil && len(field.TypeOptions.Object.Schema) > 0 {
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("must be an object")
		}

		return ValidateConfiguration(field.TypeOptions.Object.Schema, obj)
	}

	switch value.(type) {
	case map[string]any:
		return nil
	case []any:
		return nil
	default:
		return fmt.Errorf("must be an object or array")
	}
}

func validateList(field ConfigurationField, value any) error {
	list, ok := value.([]any)
	if !ok {
		return fmt.Errorf("must be a list of values")
	}

	if field.TypeOptions.List == nil {
		return nil
	}

	listOptions := field.TypeOptions.List
	if listOptions.ItemDefinition == nil {
		return nil
	}

	itemDef := listOptions.ItemDefinition
	for i, item := range list {
		if itemDef.Type == FieldTypeObject && len(itemDef.Schema) > 0 {
			itemMap, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("item at index %d must be an object", i)
			}

			err := ValidateConfiguration(itemDef.Schema, itemMap)
			if err != nil {
				return fmt.Errorf("item at index %d: %w", i, err)
			}
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
		return validateNumber(field, value)

	case FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}

	case FieldTypeSelect:
		return validateSelect(field, value)

	case FieldTypeMultiSelect:
		return validateMultiSelect(field, value)

	case FieldTypeIntegration:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeIntegrationResource:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeUser:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeRole:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeGroup:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeList:
		return validateList(field, value)

	case FieldTypeObject:
		return validateObject(field, value)
	}

	return nil
}
