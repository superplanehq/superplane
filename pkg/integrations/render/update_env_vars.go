package render

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateEnvVars struct{}

type UpdateEnvVarsConfiguration struct {
	Service string           `json:"service" mapstructure:"service"`
	EnvVars []EnvVarMapEntry `json:"envVars" mapstructure:"envVars"`
}

type EnvVarMapEntry struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

func (c *UpdateEnvVars) Name() string {
	return "render.updateEnvVars"
}

func (c *UpdateEnvVars) Label() string {
	return "Update Env Vars"
}

func (c *UpdateEnvVars) Description() string {
	return "Update environment variables for a Render service"
}

func (c *UpdateEnvVars) Documentation() string {
	return `The Update Env Vars component replaces all environment variables for a Render service.

## Use Cases

- **Config rotation**: Rotate secrets and API keys as part of a scheduled workflow
- **Environment promotion**: Copy environment variables when promoting between stages
- **Feature flags**: Toggle feature flags by updating environment variables

## Configuration

- **Service**: The Render service whose environment variables should be updated
- **Environment Variables**: List of key-value pairs to set

## Output

Returns the updated list of environment variables after the operation.

## Notes

- This operation **replaces all** environment variables for the service.
  Make sure to include all desired variables, not just the ones being changed.`
}

func (c *UpdateEnvVars) Icon() string {
	return "settings"
}

func (c *UpdateEnvVars) Color() string {
	return "gray"
}

func (c *UpdateEnvVars) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateEnvVars) Configuration() []configuration.Field {
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
			Description: "Render service to update environment variables for",
		},
		{
			Name:        "envVars",
			Label:       "Environment Variables",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Key-value pairs of environment variables to set",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "key",
								Type:        configuration.FieldTypeString,
								Label:       "Key",
								Required:    true,
								Placeholder: "MY_ENV_VAR",
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Value",
								Required:    true,
								Placeholder: "my-value",
							},
						},
					},
				},
			},
		},
	}
}

func (c *UpdateEnvVars) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdateEnvVarsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.Service == "" {
		return fmt.Errorf("service is required")
	}

	return nil
}

func (c *UpdateEnvVars) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdateEnvVarsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	envVars := make([]EnvVar, 0, len(config.EnvVars))
	for _, entry := range config.EnvVars {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}

		envVars = append(envVars, EnvVar{
			Key:   key,
			Value: entry.Value,
		})
	}

	result, err := client.UpdateEnvVars(config.Service, envVars)
	if err != nil {
		return fmt.Errorf("failed to update env vars: %w", err)
	}

	outputVars := make([]any, 0, len(result))
	for _, v := range result {
		outputVars = append(outputVars, map[string]any{
			"key":   v.Key,
			"value": v.Value,
		})
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"render.envVars.updated",
		[]any{map[string]any{
			"serviceId": config.Service,
			"envVars":   outputVars,
		}},
	)
}

func (c *UpdateEnvVars) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateEnvVars) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *UpdateEnvVars) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateEnvVars) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateEnvVars) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateEnvVars) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeUpdateEnvVarsConfiguration(configuration any) (UpdateEnvVarsConfiguration, error) {
	config := UpdateEnvVarsConfiguration{}
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return UpdateEnvVarsConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Service = strings.TrimSpace(config.Service)
	return config, nil
}
