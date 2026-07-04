package ec2

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type DeleteAlarm struct{}

type DeleteAlarmConfiguration struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarm" mapstructure:"alarm"`
}

type DeleteAlarmNodeMetadata struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarmName" mapstructure:"alarmName"`
}

func (c *DeleteAlarm) Name() string {
	return "aws.ec2.deleteAlarm"
}

func (c *DeleteAlarm) Label() string {
	return "EC2 • Delete Alarm"
}

func (c *DeleteAlarm) Description() string {
	return "Delete a CloudWatch metric alarm for an EC2 instance"
}

func (c *DeleteAlarm) Documentation() string {
	return `The Delete Alarm component removes a CloudWatch metric alarm.

## Use Cases

- **Cleanup**: Remove temporary alarms created by automation workflows
- **Decommissioning**: Delete monitoring for instances being retired
- **Rollback**: Undo alarm creation steps in failed provisioning runs

## Configuration

- **Region**: AWS region where the alarm resides
- **Alarm**: CloudWatch alarm to delete (` + "`ec2.alarm`" + ` resource picker)

## Output

Emits a deletion confirmation on the default output channel:
- ` + "`alarmName`" + `, ` + "`deleted`" + `, ` + "`region`" + `
`
}

func (c *DeleteAlarm) Icon() string {
	return "aws"
}

func (c *DeleteAlarm) Color() string {
	return "gray"
}

func (c *DeleteAlarm) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteAlarm) Configuration() []configuration.Field {
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
			Name:        "alarm",
			Label:       "Alarm",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "CloudWatch alarm to delete",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.alarm",
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

func (c *DeleteAlarm) Setup(ctx core.SetupContext) error {
	config := DeleteAlarmConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(DeleteAlarmNodeMetadata{
		Region:    region,
		AlarmName: alarmName,
	})
}

func (c *DeleteAlarm) Execute(ctx core.ExecutionContext) error {
	config := DeleteAlarmConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	if err := client.DeleteAlarms(alarmName); err != nil {
		return fmt.Errorf("failed to delete alarm: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteAlarmPayloadType,
		[]any{
			map[string]any{
				"alarmName": alarmName,
				"deleted":   true,
				"region":    region,
			},
		},
	)
}

func (c *DeleteAlarm) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteAlarm) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *DeleteAlarm) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteAlarm) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *DeleteAlarm) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *DeleteAlarm) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
