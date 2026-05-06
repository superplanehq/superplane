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

type RebootInstance struct{}

type RebootInstanceConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
}

func (c *RebootInstance) Name() string {
	return "aws.ec2.rebootInstance"
}

func (c *RebootInstance) Label() string {
	return "EC2 • Reboot Instance"
}

func (c *RebootInstance) Description() string {
	return "Reboot a running EC2 instance"
}

func (c *RebootInstance) Documentation() string {
	return `The Reboot Instance component sends a reboot request to a running EC2 instance.

Unlike stopping and starting, a reboot preserves the instance's public IP address
and does not move the instance to different underlying hardware.

## Use Cases

- **Configuration reload**: Apply OS-level changes that require a restart
- **Maintenance automation**: Reboot instances after package updates
- **Incident response**: Restart an unresponsive instance as part of a runbook

## Configuration

- **Region**: AWS region where the instance is located
- **Instance ID**: EC2 instance ID (for example: i-1234567890abcdef0)

## Output

The component emits a single event on the default channel confirming the reboot
request was accepted, including the instance ID and region.`
}

func (c *RebootInstance) Icon() string {
	return "aws"
}

func (c *RebootInstance) Color() string {
	return "gray"
}

func (c *RebootInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RebootInstance) Configuration() []configuration.Field {
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

func (c *RebootInstance) Setup(ctx core.SetupContext) error {
	config := RebootInstanceConfiguration{}
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

func (c *RebootInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RebootInstance) Execute(ctx core.ExecutionContext) error {
	config := RebootInstanceConfiguration{}
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
	output, err := client.RebootInstance(config.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to reboot instance: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.instanceReboot",
		[]any{map[string]any{
			"instanceReboot": output,
		}},
	)
}

func (c *RebootInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RebootInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RebootInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RebootInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RebootInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
