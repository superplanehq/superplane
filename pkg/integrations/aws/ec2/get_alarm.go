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

type GetAlarm struct{}

type GetAlarmConfiguration struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarm" mapstructure:"alarm"`
}

type GetAlarmNodeMetadata struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarmName" mapstructure:"alarmName"`
}

func (c *GetAlarm) Name() string {
	return "aws.ec2.getAlarm"
}

func (c *GetAlarm) Label() string {
	return "EC2 • Get Alarm"
}

func (c *GetAlarm) Description() string {
	return "Fetch the current state and details of a CloudWatch alarm for an EC2 instance"
}

func (c *GetAlarm) Documentation() string {
	return `The Get Alarm component describes a CloudWatch alarm and emits its current details.

## Use Cases

- **State inspection**: Check whether an alarm is in ALARM, OK, or INSUFFICIENT_DATA state before taking action
- **Alarm metadata lookup**: Retrieve threshold, metric, and dimension details mid-workflow
- **Audit**: Record alarm configuration at a point in time

## Configuration

- **Region**: AWS region where the alarm resides
- **Alarm**: CloudWatch alarm to describe, selected from all alarms in the chosen region (` + "`ec2.alarm`" + ` resource picker)

## Output

Emits the alarm details on the default output channel:
- ` + "`alarmName`" + `, ` + "`alarmArn`" + `, ` + "`namespace`" + `, ` + "`metricName`" + `
- ` + "`statistic`" + `, ` + "`threshold`" + `, ` + "`comparisonOperator`" + `, ` + "`stateValue`" + `
- ` + "`period`" + `, ` + "`evaluationPeriods`" + `, ` + "`dimensions`" + `, ` + "`region`" + `
`
}

func (c *GetAlarm) Icon() string {
	return "aws"
}

func (c *GetAlarm) Color() string {
	return "gray"
}

func (c *GetAlarm) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetAlarm) Configuration() []configuration.Field {
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
			Description: "CloudWatch alarm to describe",
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

func (c *GetAlarm) Setup(ctx core.SetupContext) error {
	config := GetAlarmConfiguration{}
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

	return ctx.Metadata.Set(GetAlarmNodeMetadata{
		Region:    region,
		AlarmName: alarmName,
	})
}

func (c *GetAlarm) Execute(ctx core.ExecutionContext) error {
	config := GetAlarmConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	alarmName := strings.TrimSpace(config.AlarmName)

	if region == "" {
		return fmt.Errorf("region is required")
	}
	if alarmName == "" {
		return fmt.Errorf("alarm name is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	alarm, err := client.DescribeAlarm(alarmName)
	if err != nil {
		return fmt.Errorf("failed to describe alarm: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetAlarmPayloadType,
		[]any{alarmToMap(alarm)},
	)
}

func (c *GetAlarm) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetAlarm) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *GetAlarm) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetAlarm) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *GetAlarm) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *GetAlarm) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
