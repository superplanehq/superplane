package mapper

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "mapper"

func init() {
	registry.RegisterComponent(ComponentName, &Mapper{})
}

type Spec struct {
	Fields         []FieldMapping `json:"fields" mapstructure:"fields"`
	KeepOnlyMapped bool           `json:"keepOnlyMapped" mapstructure:"keepOnlyMapped"`
}

type FieldMapping struct {
	Name  string `json:"name" mapstructure:"name"`
	Value any    `json:"value" mapstructure:"value"`
}

type Mapper struct{}

func (m *Mapper) Name() string {
	return ComponentName
}

func (m *Mapper) Label() string {
	return "Mapper"
}

func (m *Mapper) Description() string {
	return "Map fields from previous node results to a new structure"
}

func (m *Mapper) Documentation() string {
	return `The Mapper component allows you to reshape data by defining field name/value pairs that create a new data structure.

## Use Cases

- **Data transformation**: Reshape event data into a different structure
- **API preparation**: Map fields to match an expected API request format
- **Field extraction**: Pull out specific fields from complex nested data
- **Field renaming**: Rename fields from upstream nodes

## How It Works

1. Define field mappings as name/value pairs
2. Field values support ` + "`{{expression}}`" + ` syntax to reference data from previous nodes
3. Expressions are resolved automatically before execution
4. The Mapper constructs a new output object from the resolved mappings

## Field Mappings

Each mapping has:
- **Name**: The output field name (static, no expressions)
- **Value**: The value to assign (supports ` + "`{{expression}}`" + ` syntax)

## Dot Notation

Use dots in field names to create nested structures:
- ` + "`user.name`" + ` creates ` + "`{\"user\": {\"name\": \"value\"}}`" + `
- ` + "`user.profile.email`" + ` creates ` + "`{\"user\": {\"profile\": {\"email\": \"value\"}}}`" + `

## Keep Only Mapped

- When enabled (default): output contains only the explicitly mapped fields
- When disabled: output starts with the incoming event data, then mapped fields are applied on top

## Examples

- Map a GitHub push ref: Name = ` + "`branch`" + `, Value = ` + "`{{$[\"GitHub Push\"].data.ref}}`" + `
- Static value: Name = ` + "`status`" + `, Value = ` + "`active`" + `
- Nested output: Name = ` + "`user.email`" + `, Value = ` + "`{{$[\"User Lookup\"].data.email}}`" + ``
}

func (m *Mapper) Icon() string {
	return "replace"
}

func (m *Mapper) Color() string {
	return "purple"
}

func (m *Mapper) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (m *Mapper) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "fields",
			Label:       "Field Mappings",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Define the fields to map into the output",
			Default:     []map[string]any{{"name": "myField", "value": ""}},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Field",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "name",
								Label:              "Field Name",
								Type:               configuration.FieldTypeString,
								Required:           true,
								Placeholder:        "e.g. user.email",
								DisallowExpression: true,
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "e.g. {{$[\"Node\"].data.email}}",
							},
						},
					},
				},
			},
		},
		{
			Name:        "keepOnlyMapped",
			Label:       "Keep Only Mapped Fields",
			Type:        configuration.FieldTypeBool,
			Description: "When enabled, output contains only the mapped fields. When disabled, mapped fields are merged on top of the incoming data.",
			Default:     true,
		},
	}
}

func (m *Mapper) Setup(ctx core.SetupContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(spec.Fields) == 0 {
		return fmt.Errorf("at least one field mapping is required")
	}

	seen := make(map[string]bool, len(spec.Fields))
	for _, field := range spec.Fields {
		if strings.TrimSpace(field.Name) == "" {
			return fmt.Errorf("field name cannot be empty")
		}
		if seen[field.Name] {
			return fmt.Errorf("duplicate field name: %s", field.Name)
		}
		seen[field.Name] = true
	}

	return nil
}

func (m *Mapper) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Store the field mappings in metadata for auditability
	metadata := map[string]any{
		"fields":         spec.Fields,
		"keepOnlyMapped": spec.KeepOnlyMapped,
	}
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %w", err)
	}

	// Build the output map
	var output map[string]any
	if spec.KeepOnlyMapped {
		output = map[string]any{}
	} else {
		output = copyInputData(ctx.Data)
	}

	// Apply field mappings
	for _, field := range spec.Fields {
		setNestedField(output, field.Name, field.Value)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"mapper.executed",
		[]any{output},
	)
}

// setNestedField sets a value in a nested map structure using dot notation.
// For example, setNestedField(m, "user.profile.name", "Alice") creates:
// {"user": {"profile": {"name": "Alice"}}}
func setNestedField(target map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := target

	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		if existing, ok := current[key]; ok {
			if existingMap, ok := existing.(map[string]any); ok {
				current = existingMap
				continue
			}
		}
		newMap := map[string]any{}
		current[key] = newMap
		current = newMap
	}

	current[parts[len(parts)-1]] = value
}

// copyInputData creates a shallow copy of the incoming event data.
func copyInputData(data any) map[string]any {
	if data == nil {
		return map[string]any{}
	}

	inputMap, ok := data.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	result := make(map[string]any, len(inputMap))
	for k, v := range inputMap {
		result[k] = v
	}
	return result
}

func (m *Mapper) Actions() []core.Action {
	return []core.Action{}
}

func (m *Mapper) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("mapper does not support actions")
}

func (m *Mapper) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (m *Mapper) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (m *Mapper) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (m *Mapper) Cleanup(ctx core.SetupContext) error {
	return nil
}
