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

type GetInstance struct{}

type GetInstanceConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	InstanceID string `json:"instance" mapstructure:"instance"`
}

type GetInstanceNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
}

func (c *GetInstance) Name() string {
	return "aws.ec2.getInstance"
}

func (c *GetInstance) Label() string {
	return "EC2 • Get Instance"
}

func (c *GetInstance) Description() string {
	return "Fetch the current state and details of an EC2 instance"
}

func (c *GetInstance) Documentation() string {
	return `The Get Instance component describes an EC2 instance and emits its current details.

## Use Cases

- **State inspection**: Check whether an instance is running or stopped before taking action
- **IP resolution**: Retrieve the public or private IP address of an instance at runtime
- **Metadata lookup**: Fetch instance type, AMI, VPC, and tags mid-workflow

## Configuration

- **Region**: AWS region where the instance runs
- **Instance**: EC2 instance to describe

## Output

Emits the instance details on the default output channel:
- ` + "`instanceId`" + `, ` + "`state`" + `, ` + "`instanceType`" + `, ` + "`imageId`" + `
- ` + "`publicIpAddress`" + `, ` + "`privateIpAddress`" + `, ` + "`publicDnsName`" + `, ` + "`privateDnsName`" + `
- ` + "`subnetId`" + `, ` + "`vpcId`" + `, ` + "`region`" + `, ` + "`name`" + `, ` + "`launchTime`" + `
`
}

func (c *GetInstance) Icon() string {
	return "aws"
}

func (c *GetInstance) Color() string {
	return "gray"
}

func (c *GetInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetInstance) Configuration() []configuration.Field {
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
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "EC2 instance to describe",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
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

func (c *GetInstance) Setup(ctx core.SetupContext) error {
	config := GetInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(GetInstanceNodeMetadata{
		Region:       region,
		InstanceID:   instanceID,
		InstanceName: resolveInstanceName(ctx, region, instanceID),
	})
}

func (c *GetInstance) Execute(ctx core.ExecutionContext) error {
	config := GetInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}
	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	instance, err := client.DescribeInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to describe instance: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetInstancePayloadType,
		[]any{instanceDetailsToMap(instance)},
	)
}

func (c *GetInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *GetInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func resolveInstanceName(ctx core.SetupContext, region, instanceID string) string {
	if strings.TrimSpace(instanceID) == "" || ctx.HTTP == nil || ctx.Integration == nil || strings.TrimSpace(region) == "" {
		return instanceID
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return instanceID
	}

	instance, err := NewClient(ctx.HTTP, creds, region).DescribeInstance(instanceID)
	if err != nil {
		return instanceID
	}

	name := strings.TrimSpace(instance.Name)
	if name != "" {
		return name
	}

	return instanceID
}
