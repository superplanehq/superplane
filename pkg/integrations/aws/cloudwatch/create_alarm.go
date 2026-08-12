package cloudwatch

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
	Region                  string   `json:"region" mapstructure:"region"`
	AlarmName               string   `json:"alarmName" mapstructure:"alarmName"`
	Namespace               string   `json:"namespace" mapstructure:"namespace"`
	MetricName              string   `json:"metricName" mapstructure:"metricName"`
	Dimensions              string   `json:"dimensions" mapstructure:"dimensions"`
	Statistic               string   `json:"statistic" mapstructure:"statistic"`
	ComparisonOperator      string   `json:"comparisonOperator" mapstructure:"comparisonOperator"`
	Threshold               float64  `json:"threshold" mapstructure:"threshold"`
	Period                  int      `json:"period" mapstructure:"period"`
	EvaluationPeriods       int      `json:"evaluationPeriods" mapstructure:"evaluationPeriods"`
	DatapointsToAlarm       int      `json:"datapointsToAlarm" mapstructure:"datapointsToAlarm"`
	AlarmDescription        string   `json:"alarmDescription" mapstructure:"alarmDescription"`
	TreatMissingData        string   `json:"treatMissingData" mapstructure:"treatMissingData"`
	Unit                    string   `json:"unit" mapstructure:"unit"`
	ActionsEnabled          bool     `json:"actionsEnabled" mapstructure:"actionsEnabled"`
	AlarmActions            []string `json:"alarmActions" mapstructure:"alarmActions"`
	EC2Action               string   `json:"ec2Action" mapstructure:"ec2Action"`
	OKActions               []string `json:"okActions" mapstructure:"okActions"`
	InsufficientDataActions []string `json:"insufficientDataActions" mapstructure:"insufficientDataActions"`
}

type CreateAlarmNodeMetadata struct {
	Region     string `json:"region" mapstructure:"region"`
	AlarmName  string `json:"alarmName" mapstructure:"alarmName"`
	Namespace  string `json:"namespace" mapstructure:"namespace"`
	MetricName string `json:"metricName" mapstructure:"metricName"`
	Dimensions string `json:"dimensions" mapstructure:"dimensions"`
}

func (c *CreateAlarm) Name() string {
	return "aws.cloudwatch.createAlarm"
}

func (c *CreateAlarm) Label() string {
	return "CloudWatch • Create Alarm"
}

func (c *CreateAlarm) Description() string {
	return "Create a CloudWatch metric alarm for any metric"
}

func (c *CreateAlarm) Documentation() string {
	return `The Create Alarm component creates a CloudWatch metric alarm on any metric published to CloudWatch.

## Use Cases

- **Proactive monitoring**: Add alarms for new services, queues, functions or instances as they are provisioned
- **Auto-remediation**: Notify SNS topics that fan out to remediation workflows when a threshold is crossed
- **Compliance**: Guarantee every workload has the required alarms configured automatically

## Configuration

- **Region**: AWS region where the metric lives and the alarm is created
- **Alarm Name**: Unique alarm name within the region
- **Namespace**: CloudWatch namespace of the metric (e.g. ` + "`AWS/EC2`" + `, ` + "`AWS/Lambda`" + `, or a custom namespace)
- **Metric Name**: Metric to watch, listed from the selected namespace
- **Dimensions**: Dimension set that identifies the metric. Leave empty to alarm on the namespace-wide aggregate
- **Statistic**: Aggregation applied over each period (Average, Sum, Minimum, Maximum, SampleCount)
- **Comparison Operator**: How the statistic is compared to the threshold
- **Threshold**: Value the statistic is compared against
- **Period**: Evaluation window in seconds (default: 300)
- **Evaluation Periods**: Periods evaluated before the alarm changes state (default: 1)
- **Datapoints To Alarm** *(toggleable)*: Breaching datapoints required within the evaluation periods ("M out of N")
- **Alarm Description** *(toggleable)*: Free-text description
- **Treat Missing Data** *(toggleable)*: Missing data handling (missing, ignore, breaching, notBreaching)
- **Unit** *(toggleable)*: Metric unit. Leave unset unless the metric is published with several units
- **Actions Enabled**: Whether alarm actions run on state changes (default: enabled)
- **Alarm Actions** *(toggleable)*: SNS topics notified when the alarm enters ALARM
- **EC2 Action** *(toggleable)*: EC2 automation CloudWatch runs when the alarm enters ALARM — Recover, Reboot, Stop or Terminate. Shown for ` + "`AWS/EC2`" + ` alarms, and requires an InstanceId dimension
- **OK Actions** *(toggleable)*: SNS topics notified when the alarm returns to OK
- **Insufficient Data Actions** *(toggleable)*: SNS topics notified when the alarm enters INSUFFICIENT_DATA

## Output

Emits the created alarm on the default output channel, including ` + "`alarmName`" + `, ` + "`alarmArn`" + `,
` + "`namespace`" + `, ` + "`metricName`" + `, ` + "`dimensions`" + `, ` + "`statistic`" + `, ` + "`threshold`" + `,
` + "`comparisonOperator`" + `, ` + "`stateValue`" + `, the configured actions, and ` + "`consoleUrl`" + `.
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
		regionField(),
		{
			Name:     "alarmName",
			Label:    "Alarm Name",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "CloudWatch namespace of the metric to watch",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       "cloudwatch.namespace",
					Parameters: []configuration.ParameterRef{regionParameter()},
				},
			},
		},
		{
			Name:        "metricName",
			Label:       "Metric Name",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Metric to watch",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "namespace", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       "cloudwatch.metric",
					Parameters: []configuration.ParameterRef{regionParameter(), namespaceParameter()},
				},
			},
		},
		{
			Name:        "dimensions",
			Label:       "Dimensions",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Dimension set identifying the metric. Without it, the alarm watches the namespace-wide aggregate",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "metricName", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "cloudwatch.metricDimensions",
					Parameters: []configuration.ParameterRef{
						regionParameter(),
						namespaceParameter(),
						{
							Name:      "metricName",
							ValueFrom: &configuration.ParameterValueFrom{Field: "metricName"},
						},
					},
				},
			},
		},
		{
			Name:     "statistic",
			Label:    "Statistic",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  StatisticAverage,
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
			Default: defaultAlarmPeriod,
		},
		{
			Name:    "evaluationPeriods",
			Label:   "Evaluation Periods",
			Type:    configuration.FieldTypeNumber,
			Default: defaultAlarmEvaluationPeriods,
		},
		{
			Name:        "datapointsToAlarm",
			Label:       "Datapoints To Alarm",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Breaching datapoints required within the evaluation periods",
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
		unitField(),
		{
			Name:        "actionsEnabled",
			Label:       "Actions Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Run the configured actions when the alarm changes state",
		},
		alarmActionsField("alarmActions", "Alarm Actions", "SNS topics notified when the alarm enters ALARM"),
		ec2ActionField([]configuration.VisibilityCondition{
			{Field: "namespace", Values: []string{alarmNamespaceEC2}},
		}),
		alarmActionsField("okActions", "OK Actions", "SNS topics notified when the alarm returns to OK"),
		alarmActionsField(
			"insufficientDataActions",
			"Insufficient Data Actions",
			"SNS topics notified when the alarm enters INSUFFICIENT_DATA",
		),
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

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	namespace, err := requireNamespace(config.Namespace)
	if err != nil {
		return err
	}

	metricName, err := requireMetricName(config.MetricName)
	if err != nil {
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

	if err := requireDatapointsWithinEvaluationPeriods(
		config.DatapointsToAlarm,
		effectiveEvaluationPeriods(config.EvaluationPeriods),
	); err != nil {
		return err
	}

	dimensions, err := DecodeDimensions(config.Dimensions)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(CreateAlarmNodeMetadata{
		Region:     region,
		AlarmName:  alarmName,
		Namespace:  namespace,
		MetricName: metricName,
		Dimensions: FormatDimensions(dimensions),
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

	alarmName, err := requireAlarmName(config.AlarmName)
	if err != nil {
		return err
	}

	namespace, err := requireNamespace(config.Namespace)
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

	if err := requireDatapointsWithinEvaluationPeriods(
		config.DatapointsToAlarm,
		effectiveEvaluationPeriods(config.EvaluationPeriods),
	); err != nil {
		return err
	}

	dimensions, err := DecodeDimensions(config.Dimensions)
	if err != nil {
		return err
	}

	alarmActions := config.AlarmActions
	if ec2Action := strings.TrimSpace(config.EC2Action); ec2Action != "" {
		if err := requireEC2ActionTarget(namespace, dimensions); err != nil {
			return err
		}

		alarmActions = append(alarmActions, EC2AutomationARN(region, ec2Action))
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)
	err = client.PutMetricAlarm(PutMetricAlarmInput{
		AlarmName:               alarmName,
		AlarmDescription:        config.AlarmDescription,
		Namespace:               namespace,
		MetricName:              metricName,
		Dimensions:              dimensions,
		Statistic:               statistic,
		Unit:                    config.Unit,
		Period:                  config.Period,
		EvaluationPeriods:       config.EvaluationPeriods,
		DatapointsToAlarm:       config.DatapointsToAlarm,
		Threshold:               threshold,
		ComparisonOperator:      comparisonOperator,
		TreatMissingData:        config.TreatMissingData,
		ActionsEnabled:          !hasConfigKey(ctx.Configuration, "actionsEnabled") || config.ActionsEnabled,
		AlarmActions:            alarmActions,
		OKActions:               config.OKActions,
		InsufficientDataActions: config.InsufficientDataActions,
	})

	if err != nil {
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
