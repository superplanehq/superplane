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

type CreateAlarm struct{}

type CreateAlarmConfiguration struct {
	Region             string  `json:"region" mapstructure:"region"`
	InstanceID         string  `json:"instance" mapstructure:"instance"`
	AlarmName          string  `json:"alarmName" mapstructure:"alarmName"`
	MetricName         string  `json:"metricName" mapstructure:"metricName"`
	Statistic          string  `json:"statistic" mapstructure:"statistic"`
	ComparisonOperator string  `json:"comparisonOperator" mapstructure:"comparisonOperator"`
	Threshold          float64 `json:"threshold" mapstructure:"threshold"`
	Period             int     `json:"period" mapstructure:"period"`
	EvaluationPeriods  int     `json:"evaluationPeriods" mapstructure:"evaluationPeriods"`
	AlarmDescription   string  `json:"alarmDescription" mapstructure:"alarmDescription"`
	TreatMissingData   string  `json:"treatMissingData" mapstructure:"treatMissingData"`
	SNSTopicARN        string  `json:"snsTopic" mapstructure:"snsTopic"`
	AlarmAction        string  `json:"alarmAction" mapstructure:"alarmAction"`
}

type CreateAlarmNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
	AlarmName    string `json:"alarmName" mapstructure:"alarmName"`
	MetricName   string `json:"metricName" mapstructure:"metricName"`
}

func (c *CreateAlarm) Name() string {
	return "aws.ec2.createAlarm"
}

func (c *CreateAlarm) Label() string {
	return "EC2 • Create Alarm"
}

func (c *CreateAlarm) Description() string {
	return "Create a CloudWatch metric alarm scoped to an EC2 instance"
}

func (c *CreateAlarm) Documentation() string {
	return `The Create Alarm component creates a CloudWatch metric alarm targeting a specific EC2 instance.

## Use Cases

- **Proactive monitoring**: Set up CPU or network alarms as part of an instance provisioning workflow
- **Auto-remediation**: Create alarms that trigger downstream workflows when thresholds are crossed
- **Compliance**: Ensure every new instance has required alarms configured automatically

## Configuration

- **Region**: AWS region where the EC2 instance and alarm reside
- **Instance**: EC2 instance to monitor
- **Alarm Name**: Unique name for the CloudWatch alarm
- **Metric Name**: EC2 CloudWatch metric to monitor (CPU, disk, network, status checks)
- **Statistic**: Aggregation function applied over the evaluation period (Average, Sum, Min, Max, SampleCount)
- **Comparison Operator**: How the metric is compared to the threshold (e.g. GreaterThanThreshold)
- **Threshold**: Numeric value to compare the metric against
- **Period**: Evaluation window in seconds (default: 300)
- **Evaluation Periods**: Number of consecutive periods that must breach the threshold before the alarm fires (default: 1)
- **Alarm Description**: Optional free-text description
- **Treat Missing Data**: How to treat missing data points (missing, ignore, breaching, notBreaching)
- **Alarm Action** *(toggleable)*: EC2 automation action to execute when the alarm enters ALARM state — Recover, Reboot, Stop, or Terminate
- **SNS Topic (on alarm)** *(toggleable)*: SNS topic ARN to publish a notification to when the alarm enters ALARM state

## Output

Emits the created alarm details on the default output channel:
- ` + "`alarmName`" + `, ` + "`alarmArn`" + `, ` + "`namespace`" + `, ` + "`metricName`" + `
- ` + "`statistic`" + `, ` + "`threshold`" + `, ` + "`comparisonOperator`" + `, ` + "`stateValue`" + `
- ` + "`period`" + `, ` + "`evaluationPeriods`" + `, ` + "`dimensions`" + `, ` + "`region`" + `
`
}

func (c *CreateAlarm) Icon() string {
	return "aws"
}

func (c *CreateAlarm) Color() string {
	return "gray"
}

func (c *CreateAlarm) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAlarm) Configuration() []configuration.Field {
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
			Description: "EC2 instance to monitor",
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
		{
			Name:     "alarmName",
			Label:    "Alarm Name",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "metricName",
			Label:    "Metric Name",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "CPUUtilization",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: EC2MetricOptions,
				},
			},
		},
		{
			Name:     "statistic",
			Label:    "Statistic",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "Average",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmStatisticOptions,
				},
			},
		},
		{
			Name:     "comparisonOperator",
			Label:    "Comparison Operator",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "GreaterThanThreshold",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmComparisonOperatorOptions,
				},
			},
		},
		{
			Name:     "threshold",
			Label:    "Threshold",
			Type:     configuration.FieldTypeNumber,
			Required: true,
		},
		{
			Name:    "period",
			Label:   "Period (seconds)",
			Type:    configuration.FieldTypeNumber,
			Default: 300,
		},
		{
			Name:    "evaluationPeriods",
			Label:   "Evaluation Periods",
			Type:    configuration.FieldTypeNumber,
			Default: 1,
		},
		{
			Name:  "alarmDescription",
			Label: "Alarm Description",
			Type:  configuration.FieldTypeText,
		},
		{
			Name:  "treatMissingData",
			Label: "Treat Missing Data",
			Type:  configuration.FieldTypeSelect,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmTreatMissingDataOptions,
				},
			},
		},
		{
			Name:        "alarmAction",
			Label:       "Alarm Action",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Description: "EC2 action to take when the alarm enters ALARM state",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmEC2ActionOptions,
				},
			},
		},
		{
			Name:        "snsTopic",
			Label:       "SNS Topic (on alarm)",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Publish a notification to this SNS topic when the alarm enters ALARM state",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sns.topic",
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

func (c *CreateAlarm) Setup(ctx core.SetupContext) error {
	config := CreateAlarmConfiguration{}
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

	if _, err := requireAlarmName(config.AlarmName); err != nil {
		return err
	}

	if _, err := requireMetricName(config.MetricName); err != nil {
		return err
	}

	if _, err := requireStatistic(config.Statistic); err != nil {
		return err
	}

	if _, err := requireComparisonOperator(config.ComparisonOperator); err != nil {
		return err
	}

	if _, err := requireThreshold(ctx.Configuration, config.Threshold); err != nil {
		return err
	}

	return ctx.Metadata.Set(CreateAlarmNodeMetadata{
		Region:       region,
		InstanceID:   instanceID,
		InstanceName: resolveInstanceName(ctx, region, instanceID),
		AlarmName:    config.AlarmName,
		MetricName:   config.MetricName,
	})
}

func (c *CreateAlarm) Execute(ctx core.ExecutionContext) error {
	config := CreateAlarmConfiguration{}
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

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	metricName, err := requireMetricName(config.MetricName)
	if err != nil {
		return err
	}

	statistic, err := requireStatistic(config.Statistic)
	if err != nil {
		return err
	}

	comparisonOperator, err := requireComparisonOperator(config.ComparisonOperator)
	if err != nil {
		return err
	}

	threshold, err := requireThreshold(ctx.Configuration, config.Threshold)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	var alarmActions []string
	if action := strings.TrimSpace(config.AlarmAction); action != "" {
		alarmActions = append(alarmActions, fmt.Sprintf("arn:aws:automate:%s:ec2:%s", region, action))
	}
	if topic := strings.TrimSpace(config.SNSTopicARN); topic != "" {
		alarmActions = append(alarmActions, topic)
	}

	if err := client.PutMetricAlarm(PutMetricAlarmInput{
		AlarmName:          alarmName,
		AlarmDescription:   config.AlarmDescription,
		InstanceID:         instanceID,
		MetricName:         metricName,
		Statistic:          statistic,
		Period:             config.Period,
		EvaluationPeriods:  config.EvaluationPeriods,
		Threshold:          threshold,
		ComparisonOperator: comparisonOperator,
		TreatMissingData:   config.TreatMissingData,
		AlarmActions:       alarmActions,
	}); err != nil {
		return fmt.Errorf("failed to create alarm: %w", err)
	}

	alarm, err := client.DescribeAlarm(alarmName)
	if err != nil {
		return fmt.Errorf("failed to describe alarm after creation: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateAlarmPayloadType,
		[]any{alarmToMap(alarm)},
	)
}

func (c *CreateAlarm) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateAlarm) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *CreateAlarm) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAlarm) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *CreateAlarm) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *CreateAlarm) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
