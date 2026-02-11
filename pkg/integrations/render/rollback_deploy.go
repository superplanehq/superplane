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
	RollbackDeployPayloadType   = "render.rollback.deploy"
	RollbackDeployOutputChannel = "default"
)

type RollbackDeploy struct{}

type RollbackDeployConfiguration struct {
	ServiceID string `json:"serviceId" mapstructure:"serviceId"`
	DeployID  string `json:"deployId" mapstructure:"deployId"`
}

func (c *RollbackDeploy) Name() string {
	return "render.rollback_deploy"
}

func (c *RollbackDeploy) Label() string {
	return "Rollback Deploy"
}

func (c *RollbackDeploy) Description() string {
	return "Rollback a Render service to a previous deploy"
}

func (c *RollbackDeploy) Documentation() string {
	return `The Rollback Deploy component rolls back a Render service to a previous deploy.

## Use Cases

- **Quick recovery**: Rollback to a known-good deploy after a failed release
- **Automated rollback**: Trigger rollback when health checks fail after deploy

## Configuration

- **Service**: The Render service to rollback
- **Deploy ID**: The deploy ID to rollback to

## Output

Emits the new deploy object created by the rollback on the default channel.`
}

func (c *RollbackDeploy) Icon() string {
	return "rotate-ccw"
}

func (c *RollbackDeploy) Color() string {
	return "gray"
}

func (c *RollbackDeploy) ExampleOutput() map[string]any {
	return nil
}

func (c *RollbackDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: RollbackDeployOutputChannel, Label: "Default"},
	}
}

func (c *RollbackDeploy) Configuration() []configuration.Field {
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
			Description: "Render service to rollback",
		},
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Deploy ID to rollback to",
		},
	}
}

func decodeRollbackDeployConfiguration(configuration any) (RollbackDeployConfiguration, error) {
	spec := RollbackDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return RollbackDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ServiceID = strings.TrimSpace(spec.ServiceID)
	spec.DeployID = strings.TrimSpace(spec.DeployID)

	if spec.ServiceID == "" {
		return RollbackDeployConfiguration{}, fmt.Errorf("serviceId is required")
	}
	if spec.DeployID == "" {
		return RollbackDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *RollbackDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeRollbackDeployConfiguration(ctx.Configuration)
	return err
}

func (c *RollbackDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RollbackDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRollbackDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.RollbackDeploy(spec.ServiceID, spec.DeployID)
	if err != nil {
		return err
	}

	payload := deployPayloadFromDeployResponse(deploy)
	if deploy.FinishedAt != "" {
		payload["finishedAt"] = deploy.FinishedAt
	}

	return ctx.ExecutionState.Emit(RollbackDeployOutputChannel, RollbackDeployPayloadType, []any{payload})
}

func (c *RollbackDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *RollbackDeploy) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *RollbackDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (c *RollbackDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RollbackDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

// Client method for RollbackDeploy
func (cl *Client) RollbackDeploy(serviceID, deployID string) (DeployResponse, error) {
	if serviceID == "" {
		return DeployResponse{}, fmt.Errorf("serviceID is required")
	}
	if deployID == "" {
		return DeployResponse{}, fmt.Errorf("deployID is required")
	}

	_, body, err := cl.execRequestWithResponse(
		"POST",
		"/services/"+url.PathEscape(serviceID)+"/rollbacks",
		nil,
		map[string]string{"deployId": deployID},
	)
	if err != nil {
		return DeployResponse{}, err
	}

	response := DeployResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return DeployResponse{}, fmt.Errorf("failed to unmarshal rollback response: %w", err)
	}

	return response, nil
}
