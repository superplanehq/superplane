package ec2

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type StartInstance struct{}

type StartInstanceConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
}

func (c *StartInstance) Name() string {
	return "aws.ec2.startInstance"
}

func (c *StartInstance) Label() string {
	return "EC2 • Start Instance"
}

func (c *StartInstance) Description() string {
	return "Start a stopped EC2 instance"
}

func (c *StartInstance) Documentation() string {
	return `The Start Instance component sends a start request to a stopped EC2 instance.

The API call returns immediately once the start request is accepted. The instance
transitions through the **pending** state before reaching **running**.

## Use Cases

- **Scheduled scale-up**: Start instances as part of a scheduled workflow
- **Cost optimisation**: Start instances only when workloads require them
- **Incident response**: Restart a stopped instance as part of an automated runbook

## Configuration

- **Region**: AWS region where the instance is located
- **Instance ID**: EC2 instance ID (for example: i-1234567890abcdef0)

## Output

The component emits a single event on the default channel containing the instance
ID, the current state (typically **pending**), and the previous state (**stopped**).`
}

func (c *StartInstance) Icon() string {
	return "aws"
}

func (c *StartInstance) Color() string {
	return "gray"
}

func (c *StartInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *StartInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "instanceId",
			Label:       "Instance ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "i-1234567890abcdef0",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instance",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
	}
}

func (c *StartInstance) Setup(ctx core.SetupContext) error {
	config := StartInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.InstanceID = strings.TrimSpace(config.InstanceID)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}
	if config.InstanceID == "" {
		return fmt.Errorf("instance ID is required")
	}

	return nil
}

func (c *StartInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *StartInstance) Execute(ctx core.ExecutionContext) error {
	config := StartInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.InstanceID = strings.TrimSpace(config.InstanceID)

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	output, err := client.StartInstance(config.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.instanceStateChange",
		[]any{map[string]any{
			"instanceStateChange": output,
		}},
	)
}

func (c *StartInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *StartInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *StartInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *StartInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *StartInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
