// Package structuredoutput provides a guided builder for LLM structured-output
// schemas. Instead of asking users to hand-write a JSON Schema, components
// expose a list of typed fields (name, type, description, required, and nested
// sub-fields for objects/lists). The backend turns those definitions into a
// JSON Schema, applying provider-specific rules:
//
//   - Anthropic (strict=false): optional fields are omitted from "required".
//   - OpenAI strict (strict=true): every field is in "required"; optional fields
//     are made nullable instead.
//
// Every object sets "additionalProperties": false, which both providers require.
package structuredoutput

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
)

// maxNesting is how many levels of sub-fields the guided form exposes below the
// top-level field list. The configuration framework has no $ref/recursion, so
// the form must be statically unrolled to a fixed depth; 3 nested levels (4
// total) covers essentially all real structured outputs and stays within the
// renderer's proven nesting depth.
const maxNesting = 3

// Field type values offered by the builder (the "type" select).
const (
	TypeString      = "string"
	TypeNumber      = "number"
	TypeInteger     = "integer"
	TypeBoolean     = "boolean"
	TypeStringArray = "string_array"
	TypeObject      = "object"
	TypeObjectArray = "object_array"
)

// OutputField is one user-defined field in the guided builder.
type OutputField struct {
	Name        string        `json:"name" mapstructure:"name"`
	Type        string        `json:"type" mapstructure:"type"`
	Description string        `json:"description" mapstructure:"description"`
	Required    *bool         `json:"required" mapstructure:"required"`
	Fields      []OutputField `json:"fields" mapstructure:"fields"`
}

// isRequired defaults to true when the toggle was never set (required-by-default).
func isRequired(f OutputField) bool {
	return f.Required == nil || *f.Required
}

// ConfigField returns the togglable list configuration.Field for the guided
// structured-output builder.
func ConfigField(name, label, description string) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeList,
		Required:    false,
		Togglable:   true,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel:   "Field",
				Reorderable: true,
				ItemDefinition: &configuration.ListItemDefinition{
					Type:   configuration.FieldTypeObject,
					Schema: itemSchema(maxNesting),
				},
			},
		},
	}
}

func typeSelectOptions() *configuration.TypeOptions {
	return &configuration.TypeOptions{
		Select: &configuration.SelectTypeOptions{
			Options: []configuration.FieldOption{
				{Label: "String", Value: TypeString},
				{Label: "Number", Value: TypeNumber},
				{Label: "Integer", Value: TypeInteger},
				{Label: "Boolean", Value: TypeBoolean},
				{Label: "List of strings", Value: TypeStringArray},
				{Label: "Object", Value: TypeObject},
				{Label: "List of objects", Value: TypeObjectArray},
			},
		},
	}
}

// itemSchema is the per-field object schema. It appends a nested "fields" list
// (shown only for object / list-of-objects types) while there is remaining depth.
func itemSchema(remaining int) []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "field_name",
			Description: "Property name in the JSON output",
		},
		{
			Name:        "type",
			Label:       "Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     TypeString,
			TypeOptions: typeSelectOptions(),
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional hint to guide the model",
		},
		{
			Name:     "required",
			Label:    "Required",
			Type:     configuration.FieldTypeBool,
			Required: false,
			Default:  true,
		},
	}

	if remaining > 0 {
		fields = append(fields, configuration.Field{
			Name:        "fields",
			Label:       "Sub-fields",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Properties of this object (or of each item, for a list of objects)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "type", Values: []string{TypeObject, TypeObjectArray}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Field",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: itemSchema(remaining - 1),
					},
				},
			},
		})
	}

	return fields
}

// Decode converts the raw config value (a list of field maps) into []OutputField.
func Decode(raw any) ([]OutputField, error) {
	if raw == nil {
		return nil, nil
	}
	var fields []OutputField
	if err := mapstructure.Decode(raw, &fields); err != nil {
		return nil, fmt.Errorf("invalid structured output fields: %w", err)
	}
	return fields, nil
}

// Validate checks field definitions: non-empty unique names, known types, and
// that object / list-of-object fields declare at least one sub-field.
func Validate(fields []OutputField) error {
	return validate(fields, "")
}

func validate(fields []OutputField, path string) error {
	seen := map[string]bool{}
	for i, f := range fields {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			return fmt.Errorf("%sfield #%d: name is required", path, i+1)
		}
		if seen[name] {
			return fmt.Errorf("%s%q: duplicate field name", path, name)
		}
		seen[name] = true

		switch f.Type {
		case TypeString, TypeNumber, TypeInteger, TypeBoolean, TypeStringArray:
			// scalars and string lists have no sub-fields
		case TypeObject, TypeObjectArray:
			if len(f.Fields) == 0 {
				return fmt.Errorf("%s%q: a %s field requires at least one sub-field", path, name, typeLabel(f.Type))
			}
			if err := validate(f.Fields, fmt.Sprintf("%s%q -> ", path, name)); err != nil {
				return err
			}
		case "":
			return fmt.Errorf("%s%q: type is required", path, name)
		default:
			return fmt.Errorf("%s%q: unknown type %q", path, name, f.Type)
		}
	}
	return nil
}

func typeLabel(t string) string {
	switch t {
	case TypeObject:
		return "object"
	case TypeObjectArray:
		return "list of objects"
	default:
		return t
	}
}

// BuildSchema constructs a JSON Schema object from the field definitions.
func BuildSchema(fields []OutputField, strict bool) map[string]any {
	properties := map[string]any{}
	requiredNames := []string{}
	for _, f := range fields {
		properties[f.Name] = fieldSchema(f, strict)
		if strict || isRequired(f) {
			requiredNames = append(requiredNames, f.Name)
		}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             requiredNames,
		"additionalProperties": false,
	}
}

func fieldSchema(f OutputField, strict bool) map[string]any {
	var schema map[string]any
	switch f.Type {
	case TypeObject:
		schema = BuildSchema(f.Fields, strict)
	case TypeObjectArray:
		schema = map[string]any{
			"type":  "array",
			"items": BuildSchema(f.Fields, strict),
		}
	case TypeStringArray:
		schema = map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		}
	default: // string, number, integer, boolean
		schema = map[string]any{"type": f.Type}
	}

	if d := strings.TrimSpace(f.Description); d != "" {
		schema["description"] = d
	}

	// OpenAI strict mode requires every property in "required"; an optional
	// field is expressed as nullable instead.
	if strict && !isRequired(f) {
		schema = makeNullable(schema)
	}

	return schema
}

// makeNullable adds "null" to a schema's type so an optional field still
// satisfies OpenAI strict mode.
func makeNullable(schema map[string]any) map[string]any {
	switch t := schema["type"].(type) {
	case string:
		schema["type"] = []any{t, "null"}
	case []any:
		for _, v := range t {
			if v == "null" {
				return schema
			}
		}
		schema["type"] = append(t, "null")
	}
	return schema
}
