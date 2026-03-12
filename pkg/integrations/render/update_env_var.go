package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateEnvVarPayloadType = "render.envVar.updated"

const (
	updateEnvVarValueStrategySet      = "set"
	updateEnvVarValueStrategyGenerate = "generate"
)

type UpdateEnvVar struct{}

type UpdateEnvVarConfiguration struct {
	Service       string  `json:"service" mapstructure:"service"`
	Key           string  `json:"key" mapstructure:"key"`
	ValueStrategy string  `json:"valueStrategy" mapstructure:"valueStrategy"`
	Value         *string `json:"value" mapstructure:"value"`
	EmitValue     bool    `json:"emitValue" mapstructure:"emitValue"`
}

func (c *UpdateEnvVar) Name() string {
	return "render.updateEnvVar"
}

func (c *UpdateEnvVar) Label() string {
	return "Update Env Var"
}

func (c *UpdateEnvVar) Description() string {
	return "Update or generate a value for a Render service environment variable"
}

func (c *UpdateEnvVar) Documentation() string {
	return `The Update Env Var component updates a Render service environment variable.

## Use Cases

- **Rotate secrets**: Generate a new value for an env var (for example, API tokens) and optionally emit it
- **Configuration changes**: Update non-secret environment values as part of a workflow

## Configuration

- **Service**: Render service that owns the env var
- **Key**: Env var key to update
- **Value Strategy**:
  - ` + "`Set value`" + `: provide the ` + "`Value`" + ` field
  - ` + "`Generate value`" + `: request Render to generate a new value
- **Value**: New env var value (sensitive). Required when using ` + "`Set value`" + `
- **Emit Value**: When enabled, include the env var ` + "`value`" + ` in output. Disabled by default to avoid leaking secrets.

## Output

Emits a ` + "`render.envVar.updated`" + ` payload with ` + "`serviceId`" + `, ` + "`key`" + `, and a ` + "`valueGenerated`" + ` boolean. The ` + "`value`" + ` field is only included when ` + "`emitValue`" + ` is enabled.`
}

func (c *UpdateEnvVar) Icon() string {
	return "sliders-horizontal"
}

func (c *UpdateEnvVar) Color() string {
	return "gray"
}

func (c *UpdateEnvVar) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateEnvVar) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service that owns the env var",
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., DATABASE_URL",
			Description: "Env var key to update",
		},
		{
			Name:        "valueStrategy",
			Label:       "Value Strategy",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     updateEnvVarValueStrategySet,
			Description: "How to update the env var value",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Set value", Value: updateEnvVarValueStrategySet},
						{Label: "Generate value", Value: updateEnvVarValueStrategyGenerate},
					},
				},
			},
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "New env var value (only used when value strategy is set)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "valueStrategy", Values: []string{updateEnvVarValueStrategySet}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "valueStrategy", Values: []string{updateEnvVarValueStrategySet}},
			},
		},
		{
			Name:        "emitValue",
			Label:       "Emit Value",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "When enabled, include the env var value in output (disabled by default to avoid leaking secrets)",
		},
	}
}

func decodeUpdateEnvVarConfiguration(configuration any) (UpdateEnvVarConfiguration, error) {
	spec := UpdateEnvVarConfiguration{
		ValueStrategy: updateEnvVarValueStrategySet,
	}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return UpdateEnvVarConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.Key = strings.TrimSpace(spec.Key)
	spec.ValueStrategy = strings.ToLower(strings.TrimSpace(spec.ValueStrategy))
	if spec.ValueStrategy == "" {
		spec.ValueStrategy = updateEnvVarValueStrategySet
	}

	if spec.Service == "" {
		return UpdateEnvVarConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.Key == "" {
		return UpdateEnvVarConfiguration{}, fmt.Errorf("key is required")
	}

	switch spec.ValueStrategy {
	case updateEnvVarValueStrategySet:
		if spec.Value == nil {
			return UpdateEnvVarConfiguration{}, fmt.Errorf("value is required")
		}
	case updateEnvVarValueStrategyGenerate:
	default:
		return UpdateEnvVarConfiguration{}, fmt.Errorf("invalid valueStrategy: %s", spec.ValueStrategy)
	}

	return spec, nil
}

func (c *UpdateEnvVar) Setup(ctx core.SetupContext) error {
	_, err := decodeUpdateEnvVarConfiguration(ctx.Configuration)
	return err
}

func (c *UpdateEnvVar) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateEnvVar) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeUpdateEnvVarConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	request := UpdateEnvVarRequest{}
	valueGenerated := false
	switch spec.ValueStrategy {
	case updateEnvVarValueStrategyGenerate:
		valueGenerated = true
		generate := true
		request.GenerateValue = &generate
	case updateEnvVarValueStrategySet:
		request.Value = spec.Value
	}

	envVar, err := client.UpdateEnvVar(spec.Service, spec.Key, request)
	if err != nil {
		return err
	}

	key := strings.TrimSpace(envVar.Key)
	if key == "" {
		key = spec.Key
	}

	data := map[string]any{
		"serviceId":      spec.Service,
		"key":            key,
		"valueGenerated": valueGenerated,
	}

	if spec.EmitValue {
		data["value"] = envVar.Value
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateEnvVarPayloadType,
		[]any{data},
	)
}

func (c *UpdateEnvVar) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateEnvVar) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateEnvVar) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateEnvVar) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateEnvVar) Cleanup(ctx core.SetupContext) error {
	return nil
}
