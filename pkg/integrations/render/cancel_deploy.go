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
	CancelDeployPayloadType   = "render.cancel.deploy"
	CancelDeployOutputChannel = "default"
)

type CancelDeploy struct{}

type CancelDeployConfiguration struct {
	ServiceID string `json:"serviceId" mapstructure:"serviceId"`
	DeployID  string `json:"deployId" mapstructure:"deployId"`
}

func (c *CancelDeploy) Name() string {
	return "render.cancel_deploy"
}

func (c *CancelDeploy) Label() string {
	return "Cancel Deploy"
}

func (c *CancelDeploy) Description() string {
	return "Cancel a running deploy for a Render service"
}

func (c *CancelDeploy) Documentation() string {
	return `The Cancel Deploy component cancels a running deploy for a Render service.

## Use Cases

- **Abort failed deploys**: Cancel a deploy that is taking too long or known to be bad
- **Pipeline gates**: Cancel a deploy if a downstream check fails

## Configuration

- **Service**: The Render service that owns the deploy
- **Deploy ID**: The deploy ID to cancel

## Output

Emits the cancelled deploy object on the default channel.`
}

func (c *CancelDeploy) Icon() string {
	return "x-circle"
}

func (c *CancelDeploy) Color() string {
	return "gray"
}

func (c *CancelDeploy) ExampleOutput() map[string]any {
	return nil
}

func (c *CancelDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: CancelDeployOutputChannel, Label: "Default"},
	}
}

func (c *CancelDeploy) Configuration() []configuration.Field {
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
			Description: "Render deploy ID to cancel",
		},
	}
}

func decodeCancelDeployConfiguration(configuration any) (CancelDeployConfiguration, error) {
	spec := CancelDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return CancelDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ServiceID = strings.TrimSpace(spec.ServiceID)
	spec.DeployID = strings.TrimSpace(spec.DeployID)

	if spec.ServiceID == "" {
		return CancelDeployConfiguration{}, fmt.Errorf("serviceId is required")
	}
	if spec.DeployID == "" {
		return CancelDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *CancelDeploy) Setup(ctx core.SetupContext) error {
	_, err := decodeCancelDeployConfiguration(ctx.Configuration)
	return err
}

func (c *CancelDeploy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CancelDeploy) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeCancelDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.CancelDeploy(spec.ServiceID, spec.DeployID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"deployId":   deploy.ID,
		"status":     deploy.Status,
		"createdAt":  deploy.CreatedAt,
		"finishedAt": deploy.FinishedAt,
	}

	return ctx.ExecutionState.Emit(CancelDeployOutputChannel, CancelDeployPayloadType, []any{payload})
}

func (c *CancelDeploy) Actions() []core.Action {
	return []core.Action{}
}

func (c *CancelDeploy) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CancelDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (c *CancelDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CancelDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}

// Client method for CancelDeploy
func (cl *Client) CancelDeploy(serviceID, deployID string) (DeployResponse, error) {
	if serviceID == "" {
		return DeployResponse{}, fmt.Errorf("serviceID is required")
	}
	if deployID == "" {
		return DeployResponse{}, fmt.Errorf("deployID is required")
	}

	_, body, err := cl.execRequestWithResponse(
		"POST",
		"/services/"+url.PathEscape(serviceID)+"/deploys/"+url.PathEscape(deployID)+"/cancel",
		nil,
		nil,
	)
	if err != nil {
		return DeployResponse{}, err
	}

	response := DeployResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return DeployResponse{}, fmt.Errorf("failed to unmarshal cancel deploy response: %w", err)
	}

	return response, nil
}
