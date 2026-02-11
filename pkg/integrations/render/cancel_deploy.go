package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CancelDeployPayloadType          = "render.deploy.finished"
	CancelDeploySuccessOutputChannel = "success"
	CancelDeployFailedOutputChannel  = "failed"
	CancelDeployPollInterval         = 5 * time.Minute
	cancelDeployExecutionKey         = "deploy_id"
)

type CancelDeploy struct{}

type CancelDeployConfiguration struct {
	Service  string `json:"service" mapstructure:"service"`
	DeployID string `json:"deployId" mapstructure:"deployId"`
}

func (c *CancelDeploy) Name() string {
	return "render.cancelDeploy"
}

func (c *CancelDeploy) Label() string {
	return "Cancel Deploy"
}

func (c *CancelDeploy) Description() string {
	return "Cancel an in-progress deploy for a Render service and wait for it to complete"
}

func (c *CancelDeploy) Documentation() string {
	return `The Cancel Deploy component cancels an in-progress deploy for a Render service and waits for it to complete.

## Use Cases

- **Automated rollback/abort**: Cancel deploys when health checks fail
- **Manual intervention**: Stop a deploy triggered earlier in a workflow

## How It Works

1. Sends a cancel request for the specified deploy via the Render API
2. Waits for the deploy to finish (via deploy_ended webhook and optional polling fallback)
3. Routes execution based on deploy outcome:
   - **Success channel**: Deploy was cancelled successfully (status is ` + "`canceled`" + `)
   - **Failed channel**: Deploy finished with an unexpected status

## Configuration

- **Service**: Render service that owns the deploy
- **Deploy ID**: Deploy ID to cancel (supports expressions)

## Output Channels

- **Success**: Emitted when the deploy is cancelled successfully
- **Failed**: Emitted when the deploy finishes with a non-cancelled status

## Notes

- Uses the existing integration webhook for deploy_ended events
- Falls back to polling if the webhook does not arrive
- Requires a Render API key configured on the integration`
}

func (c *CancelDeploy) Icon() string {
	return "circle-slash-2"
}

func (c *CancelDeploy) Color() string {
	return "gray"
}

func (c *CancelDeploy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: CancelDeploySuccessOutputChannel, Label: "Success"},
		{Name: CancelDeployFailedOutputChannel, Label: "Failed"},
	}
}

func (c *CancelDeploy) Configuration() []configuration.Field {
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
			Description: "Render service that owns the deploy",
		},
		{
			Name:        "deployId",
			Label:       "Deploy ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., dep-... or {{$.event.data.deployId}}",
			Description: "Render deploy ID to cancel",
		},
	}
}

func decodeCancelDeployConfiguration(configuration any) (CancelDeployConfiguration, error) {
	spec := CancelDeployConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return CancelDeployConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Service = strings.TrimSpace(spec.Service)
	spec.DeployID = strings.TrimSpace(spec.DeployID)
	if spec.Service == "" {
		return CancelDeployConfiguration{}, fmt.Errorf("service is required")
	}
	if spec.DeployID == "" {
		return CancelDeployConfiguration{}, fmt.Errorf("deployId is required")
	}

	return spec, nil
}

func (c *CancelDeploy) Setup(ctx core.SetupContext) error {
	if _, err := decodeCancelDeployConfiguration(ctx.Configuration); err != nil {
		return err
	}

	ctx.Integration.RequestWebhook(webhookConfigurationForResource(
		ctx.Integration,
		webhookResourceTypeDeploy,
		[]string{"deploy_ended"},
	))

	return nil
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

	deploy, err := client.CancelDeploy(spec.Service, spec.DeployID)
	if err != nil {
		return err
	}

	deployID := readString(deploy.ID)
	if deployID == "" {
		return fmt.Errorf("cancel deploy response missing id")
	}

	err = ctx.Metadata.Set(DeployExecutionMetadata{
		Deploy: &DeployMetadata{
			ID:         deployID,
			Status:     readString(deploy.Status),
			ServiceID:  spec.Service,
			CreatedAt:  readString(deploy.CreatedAt),
			FinishedAt: readString(deploy.FinishedAt),
		},
	})
	if err != nil {
		return err
	}

	if err := ctx.ExecutionState.SetKV(cancelDeployExecutionKey, deployID); err != nil {
		return err
	}

	// If the deploy is already finished (cancel was immediate), emit right away
	if deploy.FinishedAt != "" {
		payload := deployPayloadFromDeployResponse(deploy)
		return emitDeployStatusResult(ctx.ExecutionState, deploy.Status, cancelDeployWebhookConfig(), payload)
	}

	// Wait for deploy_ended webhook; poll as fallback
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CancelDeployPollInterval)
}

func (c *CancelDeploy) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (c *CancelDeploy) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CancelDeploy) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	spec, err := decodeCancelDeployConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := DeployExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.Deploy == nil || metadata.Deploy.ID == "" {
		return nil
	}

	if metadata.Deploy.FinishedAt != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	deploy, err := client.GetDeploy(spec.Service, metadata.Deploy.ID)
	if err != nil {
		return err
	}

	if deploy.FinishedAt == "" {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, CancelDeployPollInterval)
	}

	metadata.Deploy.Status = deploy.Status
	metadata.Deploy.FinishedAt = readString(deploy.FinishedAt)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	payload := deployPayloadFromDeployResponse(deploy)
	return emitDeployStatusResult(ctx.ExecutionState, deploy.Status, cancelDeployWebhookConfig(), payload)
}

func (c *CancelDeploy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return handleDeployEndedWebhook(ctx, cancelDeployWebhookConfig())
}

func cancelDeployWebhookConfig() deployEndedWebhookConfig {
	return deployEndedWebhookConfig{
		executionKey:    cancelDeployExecutionKey,
		successStatuses: []string{"canceled"},
		successChannel:  CancelDeploySuccessOutputChannel,
		failedChannel:   CancelDeployFailedOutputChannel,
		payloadType:     CancelDeployPayloadType,
	}
}

func (c *CancelDeploy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CancelDeploy) Cleanup(ctx core.SetupContext) error {
	return nil
}
