package render

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	GetDeployPayloadType   = "render.get.deploy"
	GetDeployOutputChannel = "default"
)

type GetDeploy struct{}

type GetDeployConfiguration struct {
	ServiceID string `json:"serviceId" mapstructure:"serviceId"`
	DeployID  string `json:"deployId" mapstructure:"deployId"`
}

func (c *GetDeploy) Name() string {
	return "render.get_deploy"
}

func (c *GetDeploy) Label() string {
	return "Get Deploy"
}

func (c *GetDeploy) Description() string {
	return "Fetch details of a Render deploy by ID"
}

func (c *GetDeploy) Documentation() string {
	return `The Get Deploy component fetches details of a specific deploy for a Render service.

## Use Cases

- **Check deploy status**: Inspect the current state of a deploy
- **Retrieve deploy metadata**: Get timestamps and status for reporting

## Configuration

- **Service**: The Render service that owns the deploy
- **Deploy ID**: The deploy ID to fetch (e.g. ` + "`dep-...`" + `)

## Output

Emits the deploy object returned by the Render API on the default channel.`
}

func (c *GetDeploy) Icon() string {
	return "search"
}

func (c *GetDeploy) Color() string {
	return "gray"
}

func (c *GetDeploy) ExampleOutput() map[string]any {
	return nil
}

func (c *GetDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: GetDeployOutputChannel, Label: "Default"},
	}
}

func (c *GetDeploy) Configuration() []configuration.Field {
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
			Description: "Render service that owns the deploy",
		},
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Render deploy ID to fetch",
		},
	}
}

func decodeGetDeployConfiguration(configuration any) (GetDeployConfiguration, error) {
	spec := GetDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ServiceID = strings.TrimSpace(spec.ServiceID)
	spec.DeployID = strings.TrimSpace(spec.DeployID)

	if spec.ServiceID == "" {
		return GetDeployConfiguration{}, fmt.Errorf("serviceId is required")
	}
	if spec.DeployID == "" {
		return GetDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *GetDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeGetDeployConfiguration(ctx.Configuration)
	return err
}

func (c *GetDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.GetDeploy(spec.ServiceID, spec.DeployID)
	if err != nil {
		return err
	}

	payload := deployPayloadFromDeployResponse(deploy)

	return ctx.ExecutionState.Emit(GetDeployOutputChannel, GetDeployPayloadType, []any{payload})
}

func (c *GetDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetDeploy) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *GetDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (c *GetDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
