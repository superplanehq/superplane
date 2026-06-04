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

type UpdateAlarm struct{}

type UpdateAlarmConfiguration struct {
	Region             string  `json:"region" mapstructure:"region"`
	AlarmName          string  `json:"alarm" mapstructure:"alarm"`
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

type UpdateAlarmNodeMetadata struct {
	Region    string `json:"region" mapstructure:"region"`
	AlarmName string `json:"alarmName" mapstructure:"alarmName"`
}

func (c *UpdateAlarm) Name() string {
	return "aws.ec2.updateAlarm"
}

func (c *UpdateAlarm) Label() string {
	return "EC2 • Update Alarm"
}

func (c *UpdateAlarm) Description() string {
	return "Update an existing CloudWatch metric alarm for an EC2 instance"
}

func (c *UpdateAlarm) Documentation() string {
	return `The Update Alarm component modifies an existing CloudWatch metric alarm scoped to an EC2 instance.

## Use Cases

- **Threshold tuning**: Raise or lower alert thresholds without recreating the alarm
- **Operational changes**: Adjust evaluation periods or comparison operators as workloads change
- **Notification updates**: Add or change SNS topics and EC2 automation actions when an alarm fires

## Configuration

- **Region**: AWS region where the alarm resides
- **Alarm**: CloudWatch alarm to update (` + "`ec2.alarm`" + ` resource picker)
- **Threshold** *(toggleable)*: New numeric threshold
- **Statistic** *(toggleable)*: Aggregation function (Average, Sum, Min, Max, SampleCount)
- **Comparison Operator** *(toggleable)*: Threshold comparison operator
- **Period** *(toggleable)*: Evaluation window in seconds
- **Evaluation Periods** *(toggleable)*: Consecutive breaching periods required before ALARM
- **Alarm Description** *(toggleable)*: Free-text description
- **Treat Missing Data** *(toggleable)*: Missing data handling (missing, ignore, breaching, notBreaching)
- **Alarm Action** *(toggleable)*: EC2 automation action when the alarm enters ALARM state
- **SNS Topic (on alarm)** *(toggleable)*: SNS topic ARN to notify when the alarm enters ALARM state

At least one toggleable property must be enabled. Unspecified properties keep their current values.

## Output

Emits the updated alarm details on the default output channel (same fields as Get Alarm).
`
}

func (c *UpdateAlarm) Icon() string {
	return "aws"
}

func (c *UpdateAlarm) Color() string {
	return "gray"
}

func (c *UpdateAlarm) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateAlarm) Configuration() []configuration.Field {
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
			Description: "CloudWatch alarm to update",
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
		{
			Name:      "threshold",
			Label:     "Threshold",
			Type:      configuration.FieldTypeNumber,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "statistic",
			Label:     "Statistic",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmStatisticOptions,
				},
			},
		},
		{
			Name:      "comparisonOperator",
			Label:     "Comparison Operator",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmComparisonOperatorOptions,
				},
			},
		},
		{
			Name:      "period",
			Label:     "Period (seconds)",
			Type:      configuration.FieldTypeNumber,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "evaluationPeriods",
			Label:     "Evaluation Periods",
			Type:      configuration.FieldTypeNumber,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "alarmDescription",
			Label:     "Alarm Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "treatMissingData",
			Label:     "Treat Missing Data",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
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

func (c *UpdateAlarm) Setup(ctx core.SetupContext) error {
	config := UpdateAlarmConfiguration{}
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

	if err := requireAtLeastOneAlarmUpdate(ctx.Configuration); err != nil {
		return err
	}

	if err := validateUpdateAlarmFields(ctx.Configuration, config); err != nil {
		return err
	}

	return ctx.Metadata.Set(UpdateAlarmNodeMetadata{
		Region:    region,
		AlarmName: alarmName,
	})
}

func (c *UpdateAlarm) Execute(ctx core.ExecutionContext) error {
	config := UpdateAlarmConfiguration{}
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

	if err := requireAtLeastOneAlarmUpdate(ctx.Configuration); err != nil {
		return err
	}

	if err := validateUpdateAlarmFields(ctx.Configuration, config); err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	existing, err := client.DescribeAlarm(alarmName)
	if err != nil {
		return fmt.Errorf("failed to describe alarm: %w", err)
	}

	input, err := buildUpdateAlarmInput(region, existing, config, ctx.Configuration)
	if err != nil {
		return err
	}

	if err := client.PutMetricAlarm(input); err != nil {
		return fmt.Errorf("failed to update alarm: %w", err)
	}

	alarm, err := client.DescribeAlarm(alarmName)
	if err != nil {
		return fmt.Errorf("failed to describe alarm after update: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateAlarmPayloadType,
		[]any{alarmToMap(alarm)},
	)
}

func validateUpdateAlarmFields(rawConfiguration any, config UpdateAlarmConfiguration) error {
	if hasConfigKey(rawConfiguration, "statistic") {
		if _, err := requireStatistic(config.Statistic); err != nil {
			return err
		}
	}

	if hasConfigKey(rawConfiguration, "comparisonOperator") {
		if _, err := requireComparisonOperator(config.ComparisonOperator); err != nil {
			return err
		}
	}

	if hasConfigKey(rawConfiguration, "threshold") {
		if _, err := requireThreshold(rawConfiguration, config.Threshold); err != nil {
			return err
		}
	}

	return nil
}

func buildUpdateAlarmInput(
	region string,
	existing *MetricAlarm,
	config UpdateAlarmConfiguration,
	rawConfiguration any,
) (PutMetricAlarmInput, error) {
	instanceID, err := instanceIDFromAlarm(existing)
	if err != nil {
		return PutMetricAlarmInput{}, err
	}

	if strings.TrimSpace(existing.Namespace) != "" && existing.Namespace != alarmNamespaceEC2 {
		return PutMetricAlarmInput{}, fmt.Errorf("alarm %q is not an EC2 metric alarm", existing.AlarmName)
	}

	input := PutMetricAlarmInput{
		AlarmName:          existing.AlarmName,
		AlarmDescription:   existing.AlarmDescription,
		InstanceID:         instanceID,
		MetricName:         existing.MetricName,
		Statistic:          existing.Statistic,
		Period:             existing.Period,
		EvaluationPeriods:  existing.EvaluationPeriods,
		Threshold:          existing.Threshold,
		ComparisonOperator: existing.ComparisonOperator,
		TreatMissingData:   existing.TreatMissingData,
		OmitAlarmActions:   true,
	}

	if hasConfigKey(rawConfiguration, "alarmDescription") {
		input.AlarmDescription = config.AlarmDescription
	}

	if hasConfigKey(rawConfiguration, "statistic") {
		input.Statistic = strings.TrimSpace(config.Statistic)
	}

	if hasConfigKey(rawConfiguration, "comparisonOperator") {
		input.ComparisonOperator = strings.TrimSpace(config.ComparisonOperator)
	}

	if hasConfigKey(rawConfiguration, "threshold") {
		input.Threshold = config.Threshold
	}

	if hasConfigKey(rawConfiguration, "period") && config.Period > 0 {
		input.Period = config.Period
	}

	if hasConfigKey(rawConfiguration, "evaluationPeriods") && config.EvaluationPeriods > 0 {
		input.EvaluationPeriods = config.EvaluationPeriods
	}

	if hasConfigKey(rawConfiguration, "treatMissingData") {
		input.TreatMissingData = config.TreatMissingData
	}

	if hasConfigKey(rawConfiguration, "alarmAction") || hasConfigKey(rawConfiguration, "snsTopic") {
		input.OmitAlarmActions = false
		var alarmActions []string
		if action := strings.TrimSpace(config.AlarmAction); action != "" {
			alarmActions = append(alarmActions, fmt.Sprintf("arn:aws:automate:%s:ec2:%s", region, action))
		}
		if topic := strings.TrimSpace(config.SNSTopicARN); topic != "" {
			alarmActions = append(alarmActions, topic)
		}
		input.AlarmActions = alarmActions
	}

	return input, nil
}

func instanceIDFromAlarm(alarm *MetricAlarm) (string, error) {
	for _, dimension := range alarm.Dimensions {
		if dimension.Name == "InstanceId" && strings.TrimSpace(dimension.Value) != "" {
			return strings.TrimSpace(dimension.Value), nil
		}
	}

	return "", fmt.Errorf("alarm %q has no InstanceId dimension", alarm.AlarmName)
}

func (c *UpdateAlarm) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateAlarm) HandleHook(_ core.ActionHookContext) error {
	return nil
}

func (c *UpdateAlarm) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateAlarm) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *UpdateAlarm) Cleanup(_ core.SetupContext) error {
	return nil
}

func (c *UpdateAlarm) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
