package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateServicePayloadType = "render.service.updated"

type UpdateService struct{}

type UpdateServiceConfiguration struct {
	Service    string `json:"service" mapstructure:"service"`
	AutoDeploy string `json:"autoDeploy" mapstructure:"autoDeploy"`
}

func (c *UpdateService) Name() string { return "render.updateService" }

func (c *UpdateService) Label() string { return "Update Service" }

func (c *UpdateService) Description() string {
	return "Update Render service settings such as auto deploy"
}

func (c *UpdateService) Documentation() string {
	return `Update Render service settings. Use autoDeploy=no to freeze deploys during an incident and autoDeploy=yes to thaw them.`
}

func (c *UpdateService) Icon() string { return "settings" }

func (c *UpdateService) Color() string { return "gray" }

func (c *UpdateService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateService) Configuration() []configuration.Field {
	return []configuration.Field{
		serviceField("Render service to update"),
		{
			Name:        "autoDeploy",
			Label:       "Auto Deploy",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "no",
			Description: "Set yes to thaw deploys or no to freeze deploys",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Enabled", Value: "yes"},
						{Label: "Disabled", Value: "no"},
					},
				},
			},
		},
	}
}

func decodeUpdateServiceConfiguration(configuration any) (UpdateServiceConfiguration, error) {
	spec := UpdateServiceConfiguration{}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return UpdateServiceConfiguration{}, err
	}
	spec.Service = strings.TrimSpace(spec.Service)
	spec.AutoDeploy = strings.TrimSpace(spec.AutoDeploy)
	if spec.Service == "" {
		return UpdateServiceConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.AutoDeploy != "yes" && spec.AutoDeploy != "no" {
		return UpdateServiceConfiguration{}, fmt.Errorf("autoDeploy must be yes or no")
	}
	return spec, nil
}

func (c *UpdateService) Setup(ctx core.SetupContext) error {
	_, err := decodeUpdateServiceConfiguration(ctx.Configuration)
	return err
}

func (c *UpdateService) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeUpdateServiceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	service, err := client.UpdateService(spec.Service, spec.AutoDeploy)
	if err != nil {
		return err
	}

	data := serviceDataFromService(service)
	data["serviceId"] = spec.Service
	data["autoDeploy"] = spec.AutoDeploy
	data["status"] = "updated"

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UpdateServicePayloadType, []any{data})
}

func (c *UpdateService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *UpdateService) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *UpdateService) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *UpdateService) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *UpdateService) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *UpdateService) HandleHook(ctx core.ActionHookContext) error { return nil }
