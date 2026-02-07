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

const TriggerDeployPayloadType = "render.deploy.triggered"

type TriggerDeploy struct{}

type TriggerDeployConfiguration struct {
	Service    string `json:"service" mapstructure:"service"`
	ClearCache bool   `json:"clearCache" mapstructure:"clearCache"`
}

func (c *TriggerDeploy) Name() string {
	return "render.triggerDeploy"
}

func (c *TriggerDeploy) Label() string {
	return "Trigger Deploy"
}

func (c *TriggerDeploy) Description() string {
	return "Trigger a deploy for a Render service"
}

func (c *TriggerDeploy) Documentation() string {
	return `The Trigger Deploy component starts a new deploy for a Render service via the Render API.

## Use Cases

- **Merge to deploy**: Trigger production deploys after a successful GitHub merge and CI pass
- **Scheduled redeploys**: Redeploy staging services on schedules or external content changes
- **Chained deploys**: Deploy service B when service A finishes successfully

## Configuration

- **Service**: Render service to deploy
- **Clear Cache**: Clear build cache before deploying

## Output

The default output emits the deploy object returned by Render (e.g. ` + "`id`" + `, ` + "`status`" + `, ` + "`createdAt`" + `, ` + "`finishedAt`" + ` when available).

## Notes

- Requires a Render API key configured on the integration`
}

func (c *TriggerDeploy) Icon() string {
	return "rocket"
}

func (c *TriggerDeploy) Color() string {
	return "gray"
}

func (c *TriggerDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *TriggerDeploy) Configuration() []configuration.Field {
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
			Description: "Render service to deploy",
		},
		{
			Name:        "clearCache",
			Label:       "Clear Cache",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Clear build cache before triggering the deploy",
		},
	}
}

func (c *TriggerDeploy) Setup(ctx core.SetupContext) error {
	configuration := TriggerDeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &configuration); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(configuration.Service) == "" {
		return fmt.Errorf("service is required")
	}

	return nil
}

func (c *TriggerDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TriggerDeploy) Execute(ctx core.ExecutionContext) error {
	configuration := TriggerDeployConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &configuration); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(configuration.Service) == "" {
		return fmt.Errorf("service is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.TriggerDeploy(configuration.Service, configuration.ClearCache)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, TriggerDeployPayloadType, []any{deploy})
}

func (c *TriggerDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *TriggerDeploy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *TriggerDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *TriggerDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TriggerDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
