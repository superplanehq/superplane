package render

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	UpdateEnvVarPayloadType   = "render.update.env_var"
	UpdateEnvVarOutputChannel = "default"
)

type UpdateEnvVar struct{}

type UpdateEnvVarConfiguration struct {
	ServiceID string `json:"serviceId" mapstructure:"serviceId"`
	Key       string `json:"key" mapstructure:"key"`
	Value     string `json:"value" mapstructure:"value"`
}

type EnvVarResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *UpdateEnvVar) Name() string {
	return "render.updateEnvVar"
}

func (c *UpdateEnvVar) Label() string {
	return "Update Env Var"
}

func (c *UpdateEnvVar) Description() string {
	return "Update an environment variable for a Render service"
}

func (c *UpdateEnvVar) Documentation() string {
	return `The Update Env Var component updates an environment variable on a Render service.

## Use Cases

- **Configuration updates**: Change feature flags or config values without redeploying
- **Secret rotation**: Update secrets or API keys on services
- **Environment promotion**: Copy env var values between environments

## Configuration

- **Service**: The Render service to update
- **Key**: The environment variable key
- **Value**: The new value to set

## Output

Emits the updated environment variable key and value on the default channel.`
}

func (c *UpdateEnvVar) Icon() string {
	return "settings"
}

func (c *UpdateEnvVar) Color() string {
	return "gray"
}

func (c *UpdateEnvVar) ExampleOutput() map[string]any {
	return nil
}

func (c *UpdateEnvVar) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: UpdateEnvVarOutputChannel, Label: "Default"},
	}
}

func (c *UpdateEnvVar) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "serviceId",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
			Description: "Render service to update",
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Environment variable key",
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Environment variable value",
		},
	}
}

func decodeUpdateEnvVarConfiguration(configuration any) (UpdateEnvVarConfiguration, error) {
	spec := UpdateEnvVarConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return UpdateEnvVarConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ServiceID = strings.TrimSpace(spec.ServiceID)
	spec.Key = strings.TrimSpace(spec.Key)

	if spec.ServiceID == "" {
		return UpdateEnvVarConfiguration{}, fmt.Errorf("serviceId is required")
	}
	if spec.Key == "" {
		return UpdateEnvVarConfiguration{}, fmt.Errorf("key is required")
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

	envVar, err := client.UpdateEnvVar(spec.ServiceID, spec.Key, spec.Value)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"serviceId": spec.ServiceID,
		"key":       envVar.Key,
		"value":     envVar.Value,
	}

	return ctx.ExecutionState.Emit(UpdateEnvVarOutputChannel, UpdateEnvVarPayloadType, []any{payload})
}

func (c *UpdateEnvVar) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateEnvVar) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *UpdateEnvVar) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (c *UpdateEnvVar) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateEnvVar) Cleanup(ctx core.SetupContext) error {
	return nil
}

// Client method for UpdateEnvVar
func (cl *Client) UpdateEnvVar(serviceID, key, value string) (EnvVarResponse, error) {
	if serviceID == "" {
		return EnvVarResponse{}, fmt.Errorf("serviceID is required")
	}
	if key == "" {
		return EnvVarResponse{}, fmt.Errorf("key is required")
	}

	_, body, err := cl.execRequestWithResponse(
		"PUT",
		"/services/"+url.PathEscape(serviceID)+"/env-vars/"+url.PathEscape(key),
		nil,
		map[string]string{"value": value},
	)
	if err != nil {
		return EnvVarResponse{}, err
	}

	response := EnvVarResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return EnvVarResponse{}, fmt.Errorf("failed to unmarshal env var response: %w", err)
	}

	return response, nil
}
