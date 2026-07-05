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

type GetInstanceStatus struct{}

type GetInstanceStatusConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
}

func (c *GetInstanceStatus) Name() string {
	return "aws.ec2.getInstanceStatus"
}

func (c *GetInstanceStatus) Label() string {
	return "EC2 • Get Instance Status"
}

func (c *GetInstanceStatus) Description() string {
	return "Get the current status and health of an EC2 instance"
}

func (c *GetInstanceStatus) Documentation() string {
	return `The Get Instance Status component retrieves the current state and system health checks of an EC2 instance.

## Use Cases

- **Deployment gates**: Verify an instance is running and healthy before routing traffic
- **Monitoring workflows**: Poll instance status after a start or stop operation
- **Incident response**: Check instance health as part of an automated runbook

## Configuration

- **Region**: AWS region where the instance is located
- **Instance ID**: EC2 instance ID (for example: i-1234567890abcdef0)

## Output

The component emits a single event on the default channel containing:
- **instanceId**: The EC2 instance ID
- **instanceState**: Current lifecycle state (pending, running, stopping, stopped, ...)
- **systemStatus**: AWS system-level health check result (ok, impaired, ...)
- **instanceCheck**: Instance reachability check result (ok, impaired, ...)
- **availabilityZone**: Availability zone of the instance`
}

func (c *GetInstanceStatus) Icon() string {
	return "aws"
}

func (c *GetInstanceStatus) Color() string {
	return "gray"
}

func (c *GetInstanceStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetInstanceStatus) Configuration() []configuration.Field {
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

func (c *GetInstanceStatus) Setup(ctx core.SetupContext) error {
	config := GetInstanceStatusConfiguration{}
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

func (c *GetInstanceStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetInstanceStatus) Execute(ctx core.ExecutionContext) error {
	config := GetInstanceStatusConfiguration{}
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
	status, err := client.GetInstanceStatusCheck(config.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to describe instance status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.instanceStatus",
		[]any{map[string]any{
			"instanceStatus": status,
		}},
	)
}

func (c *GetInstanceStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetInstanceStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetInstanceStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetInstanceStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetInstanceStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
