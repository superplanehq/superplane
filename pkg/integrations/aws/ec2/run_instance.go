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

type RunInstance struct{}

type RunInstanceConfiguration struct {
	Region          string `json:"region" mapstructure:"region"`
	ImageID         string `json:"imageId" mapstructure:"imageId"`
	InstanceType    string `json:"instanceType" mapstructure:"instanceType"`
	KeyName         string `json:"keyName" mapstructure:"keyName"`
	SubnetID        string `json:"subnetId" mapstructure:"subnetId"`
	SecurityGroupID string `json:"securityGroupId" mapstructure:"securityGroupId"`
	UserData        string `json:"userData" mapstructure:"userData"`
}

func (c *RunInstance) Name() string {
	return "aws.ec2.runInstance"
}

func (c *RunInstance) Label() string {
	return "EC2 • Run Instance"
}

func (c *RunInstance) Description() string {
	return "Launch a new EC2 instance from an AMI"
}

func (c *RunInstance) Documentation() string {
	return `The Run Instance component launches a new EC2 instance from a specified AMI.

The API call returns immediately once the instance is accepted into the **pending**
state. Use the **Get Instance Status** component to poll for the **running** state
if downstream steps depend on the instance being fully started.

## Use Cases

- **Dynamic scaling**: Launch instances on demand as part of a workflow
- **Ephemeral workloads**: Start a compute instance for a specific job, then stop it
- **Blue/green deployments**: Launch a new instance before decommissioning the old one

## Configuration

- **Region**: AWS region to launch the instance in
- **Image ID**: AMI to launch the instance from (for example: ami-1234567890abcdef0)
- **Instance Type**: EC2 instance type (for example: t3.micro)
- **Key Name** *(optional)*: SSH key pair name for the instance
- **Subnet ID** *(optional)*: VPC subnet to launch the instance into
- **Security Group ID** *(optional)*: Security group to attach to the instance
- **User Data** *(optional)*: Bootstrap script to run on first launch

## Output

The component emits a single event on the default channel containing the new
instance ID, instance type, initial state (**pending**), and region.`
}

func (c *RunInstance) Icon() string {
	return "aws"
}

func (c *RunInstance) Color() string {
	return "gray"
}

func (c *RunInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RunInstance) Configuration() []configuration.Field {
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
			Name:        "imageId",
			Label:       "Image ID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "ami-1234567890abcdef0",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.image",
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
		{
			Name:        "instanceType",
			Label:       "Instance Type",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "t3.micro",
			Placeholder: "t3.micro",
		},
		{
			Name:        "keyName",
			Label:       "Key Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "my-key-pair",
		},
		{
			Name:        "subnetId",
			Label:       "Subnet ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "subnet-1234567890abcdef0",
		},
		{
			Name:        "securityGroupId",
			Label:       "Security Group ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "sg-1234567890abcdef0",
		},
		{
			Name:        "userData",
			Label:       "User Data",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "#!/bin/bash\necho hello",
		},
	}
}

func (c *RunInstance) Setup(ctx core.SetupContext) error {
	config := RunInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.ImageID = strings.TrimSpace(config.ImageID)
	config.InstanceType = strings.TrimSpace(config.InstanceType)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}
	if config.ImageID == "" {
		return fmt.Errorf("image ID is required")
	}
	if config.InstanceType == "" {
		return fmt.Errorf("instance type is required")
	}

	return nil
}

func (c *RunInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunInstance) Execute(ctx core.ExecutionContext) error {
	config := RunInstanceConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	var securityGroupIDs []string
	if sg := strings.TrimSpace(config.SecurityGroupID); sg != "" {
		securityGroupIDs = []string{sg}
	}

	client := NewClient(ctx.HTTP, creds, strings.TrimSpace(config.Region))
	output, err := client.RunInstance(RunInstanceInput{
		ImageID:          strings.TrimSpace(config.ImageID),
		InstanceType:     strings.TrimSpace(config.InstanceType),
		KeyName:          config.KeyName,
		SubnetID:         config.SubnetID,
		SecurityGroupIDs: securityGroupIDs,
		UserData:         config.UserData,
	})
	if err != nil {
		return fmt.Errorf("failed to run instance: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ec2.instance",
		[]any{map[string]any{
			"instance": output,
		}},
	)
}

func (c *RunInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RunInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RunInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RunInstance) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
