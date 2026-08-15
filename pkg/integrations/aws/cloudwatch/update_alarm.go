package cloudwatch

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type UpdateAlarm struct{}

type UpdateAlarmConfiguration struct {
	Region                  string             `json:"region" mapstructure:"region"`
	AlarmName               string             `json:"alarm" mapstructure:"alarm"`
	ThresholdCondition      ThresholdCondition `json:"thresholdCondition" mapstructure:"thresholdCondition"`
	Statistic               string             `json:"statistic" mapstructure:"statistic"`
	Period                  int                `json:"period" mapstructure:"period"`
	EvaluationPeriods       int                `json:"evaluationPeriods" mapstructure:"evaluationPeriods"`
	DatapointsToAlarm       int                `json:"datapointsToAlarm" mapstructure:"datapointsToAlarm"`
	AlarmDescription        string             `json:"alarmDescription" mapstructure:"alarmDescription"`
	TreatMissingData        string             `json:"treatMissingData" mapstructure:"treatMissingData"`
	Unit                    string             `json:"unit" mapstructure:"unit"`
	ActionsEnabled          string             `json:"actionsEnabled" mapstructure:"actionsEnabled"`
	AlarmActions            []string           `json:"alarmActions" mapstructure:"alarmActions"`
	EC2Action               string             `json:"ec2Action" mapstructure:"ec2Action"`
	OKActions               []string           `json:"okActions" mapstructure:"okActions"`
	InsufficientDataActions []string           `json:"insufficientDataActions" mapstructure:"insufficientDataActions"`
}

type ThresholdCondition struct {
	Threshold          float64 `json:"threshold" mapstructure:"threshold"`
	ComparisonOperator string  `json:"comparisonOperator" mapstructure:"comparisonOperator"`
}

type UpdateAlarmNodeMetadata struct {
	Region        string   `json:"region" mapstructure:"region"`
	AlarmName     string   `json:"alarmName" mapstructure:"alarmName"`
	UpdatedFields []string `json:"updatedFields" mapstructure:"updatedFields"`
}

func (c *UpdateAlarm) Name() string {
	return "aws.cloudwatch.updateAlarm"
}

func (c *UpdateAlarm) Label() string {
	return "CloudWatch • Update Alarm"
}

func (c *UpdateAlarm) Description() string {
	return "Update an existing CloudWatch metric alarm"
}

func (c *UpdateAlarm) Documentation() string {
	return `The Update Alarm component modifies an existing CloudWatch metric alarm.

Only the properties you enable are changed; everything else is read from the current alarm and
sent back unchanged, because CloudWatch replaces the whole alarm configuration on every update.

## Use Cases

- **Threshold tuning**: Raise or lower thresholds without recreating the alarm
- **Operational changes**: Adjust periods, statistics or missing-data handling as workloads change
- **Notification updates**: Change which SNS topics are notified on each state transition
- **Self-healing**: Attach or change an EC2 recover/reboot action on an existing alarm
- **Maintenance windows**: Disable alarm actions during planned work and re-enable them afterwards

## Configuration

- **Region**: AWS region where the alarm lives
- **Alarm**: CloudWatch alarm to update (` + "`cloudwatch.alarm`" + ` resource picker)
- **Threshold** *(toggleable)*: New threshold and comparison operator (both set together)
- **Statistic** *(toggleable)*: Aggregation applied over each period
- **Period** *(toggleable)*: Evaluation window in seconds
- **Evaluation Periods** *(toggleable)*: Periods evaluated before the alarm changes state
- **Datapoints To Alarm** *(toggleable)*: Breaching datapoints required within the evaluation periods
- **Alarm Description** *(toggleable)*: Free-text description
- **Treat Missing Data** *(toggleable)*: Missing data handling (missing, ignore, breaching, notBreaching)
- **Unit** *(toggleable)*: Metric unit. Choose **No unit** to remove it, which is how you recover an alarm whose unit matches no datapoints. This is not the same as the literal **None** unit, which still filters datapoints
- **Actions Enabled** *(toggleable)*: Whether alarm actions run on state changes
- **Alarm Actions** *(toggleable)*: SNS topics notified when the alarm enters ALARM
- **EC2 Action** *(toggleable)*: EC2 automation CloudWatch runs when the alarm enters ALARM. Only valid on an ` + "`AWS/EC2`" + ` alarm with an InstanceId dimension
- **OK Actions** *(toggleable)*: SNS topics notified when the alarm returns to OK
- **Insufficient Data Actions** *(toggleable)*: SNS topics notified when the alarm enters INSUFFICIENT_DATA

At least one toggleable property must be enabled. Alarms built on metric math, anomaly detection or
Metrics Insights queries cannot be updated by this component.

## Output

Emits the updated alarm on the default output channel, with the same fields as Create Alarm.
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
		regionField(),
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
					Type:       "cloudwatch.alarm",
					Parameters: []configuration.ParameterRef{regionParameter()},
				},
			},
		},
		{
			Name:        "thresholdCondition",
			Label:       "Threshold",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Update the alarm threshold and comparison operator",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "threshold",
							Label:    "Threshold",
							Type:     configuration.FieldTypeNumber,
							Required: true,
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
					},
				},
			},
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
		unitField(true),
		{
			Name:        "actionsEnabled",
			Label:       "Actions Enabled",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     "true",
			Description: "Run the configured actions when the alarm changes state",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Enabled", Value: "true"},
						{Label: "Disabled", Value: "false"},
					},
				},
			},
		},
		alarmActionsField("alarmActions", "Alarm Actions", "SNS topics notified when the alarm enters ALARM"),
		ec2ActionField(nil),
		alarmActionsField("okActions", "OK Actions", "SNS topics notified when the alarm returns to OK"),
		alarmActionsField(
			"insufficientDataActions",
			"Insufficient Data Actions",
			"SNS topics notified when the alarm enters INSUFFICIENT_DATA",
		),
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
		Region:        region,
		AlarmName:     alarmName,
		UpdatedFields: updatedAlarmFieldLabels(ctx.Configuration),
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
	if hasConfigKey(rawConfiguration, "thresholdCondition") {
		conditionConfig, ok := thresholdConditionConfig(rawConfiguration)
		if !ok {
			return fmt.Errorf("threshold and comparison operator are required")
		}

		if _, err := requireThreshold(conditionConfig, config.ThresholdCondition.Threshold); err != nil {
			return err
		}

		if _, err := requireComparisonOperator(config.ThresholdCondition.ComparisonOperator); err != nil {
			return err
		}
	}

	if hasConfigKey(rawConfiguration, "statistic") {
		if _, err := requireStatistic(config.Statistic); err != nil {
			return err
		}
	}

	if hasConfigKey(rawConfiguration, "treatMissingData") {
		if _, err := requireTreatMissingData(config.TreatMissingData); err != nil {
			return err
		}
	}

	// unit is deliberately not required: the "No unit" option clears the alarm's
	// unit, which is the only way back from a unit that matches no datapoints.

	if hasConfigKey(rawConfiguration, "period") && config.Period <= 0 {
		return fmt.Errorf("period must be greater than 0")
	}

	if hasConfigKey(rawConfiguration, "evaluationPeriods") && config.EvaluationPeriods <= 0 {
		return fmt.Errorf("evaluation periods must be greater than 0")
	}

	if hasConfigKey(rawConfiguration, "datapointsToAlarm") && config.DatapointsToAlarm <= 0 {
		return fmt.Errorf("datapoints to alarm must be greater than 0")
	}

	return nil
}

func thresholdConditionConfig(rawConfiguration any) (map[string]any, bool) {
	configurationMap, ok := rawConfiguration.(map[string]any)
	if !ok {
		return nil, false
	}

	conditionConfig, ok := configurationMap["thresholdCondition"].(map[string]any)
	if !ok {
		return nil, false
	}

	return conditionConfig, true
}

func buildUpdateAlarmInput(
	region string,
	existing *MetricAlarm,
	config UpdateAlarmConfiguration,
	rawConfiguration any,
) (PutMetricAlarmInput, error) {
	// A Metrics array or a threshold metric id means the alarm is driven by a
	// query or an anomaly model; an empty metric name means it is neither. Any of
	// the three would be flattened into a plain threshold alarm by PutMetricAlarm.
	if existing.HasMetricQueries ||
		strings.TrimSpace(existing.ThresholdMetricID) != "" ||
		strings.TrimSpace(existing.MetricName) == "" {
		return PutMetricAlarmInput{}, fmt.Errorf(
			"alarm %q is not a single-metric alarm and cannot be updated by this component",
			existing.AlarmName,
		)
	}

	// Seed with the current alarm: PutMetricAlarm overwrites the whole configuration.
	input := PutMetricAlarmInput{
		AlarmName:               existing.AlarmName,
		AlarmDescription:        existing.AlarmDescription,
		Namespace:               existing.Namespace,
		MetricName:              existing.MetricName,
		Dimensions:              existing.Dimensions,
		Statistic:               existing.Statistic,
		ExtendedStatistic:       existing.ExtendedStatistic,
		Unit:                    existing.Unit,
		Period:                  existing.Period,
		EvaluationPeriods:       existing.EvaluationPeriods,
		DatapointsToAlarm:       existing.DatapointsToAlarm,
		Threshold:               existing.Threshold,
		ComparisonOperator:      existing.ComparisonOperator,
		TreatMissingData:        existing.TreatMissingData,
		ActionsEnabled:          existing.ActionsEnabled,
		AlarmActions:            existing.AlarmActions,
		OKActions:               existing.OKActions,
		InsufficientDataActions: existing.InsufficientDataActions,

		EvaluateLowSampleCountPercentile: existing.EvaluateLowSampleCountPercentile,
	}

	if hasConfigKey(rawConfiguration, "alarmDescription") {
		input.AlarmDescription = config.AlarmDescription
		input.IncludeAlarmDescription = true
	}

	// Switching to a plain statistic drops the percentile-only settings with it.
	if hasConfigKey(rawConfiguration, "statistic") {
		input.Statistic = strings.TrimSpace(config.Statistic)
		input.ExtendedStatistic = ""
		input.EvaluateLowSampleCountPercentile = ""
	}

	if conditionConfig, ok := thresholdConditionConfig(rawConfiguration); ok {
		if _, err := requireThreshold(conditionConfig, config.ThresholdCondition.Threshold); err != nil {
			return PutMetricAlarmInput{}, err
		}

		comparisonOperator, err := requireComparisonOperator(config.ThresholdCondition.ComparisonOperator)
		if err != nil {
			return PutMetricAlarmInput{}, err
		}

		input.Threshold = config.ThresholdCondition.Threshold
		input.ComparisonOperator = comparisonOperator
	}

	if hasConfigKey(rawConfiguration, "period") {
		input.Period = config.Period
	}

	if hasConfigKey(rawConfiguration, "evaluationPeriods") {
		input.EvaluationPeriods = config.EvaluationPeriods
	}

	if hasConfigKey(rawConfiguration, "datapointsToAlarm") {
		input.DatapointsToAlarm = config.DatapointsToAlarm
	}

	// Shrinking the evaluation periods below a datapoints-to-alarm the user never
	// touched would push the alarm past CloudWatch's "M out of N" rule, so the
	// inherited M follows N down. An M they set themselves is their call and errors.
	if input.DatapointsToAlarm > effectiveEvaluationPeriods(input.EvaluationPeriods) {
		if hasConfigKey(rawConfiguration, "datapointsToAlarm") {
			return PutMetricAlarmInput{}, requireDatapointsWithinEvaluationPeriods(
				input.DatapointsToAlarm,
				effectiveEvaluationPeriods(input.EvaluationPeriods),
			)
		}

		input.DatapointsToAlarm = effectiveEvaluationPeriods(input.EvaluationPeriods)
	}

	if hasConfigKey(rawConfiguration, "treatMissingData") {
		input.TreatMissingData = strings.TrimSpace(config.TreatMissingData)
	}

	if hasConfigKey(rawConfiguration, "unit") {
		input.Unit = resolveUnit(config.Unit)
	}

	if hasConfigKey(rawConfiguration, "actionsEnabled") {
		input.ActionsEnabled = strings.TrimSpace(config.ActionsEnabled) != "false"
	}

	// The ALARM transition carries both SNS topics and the EC2 action, so only the
	// kinds the user enabled are replaced. An enabled-but-empty field clears its kind.
	replaceSNS := hasConfigKey(rawConfiguration, "alarmActions")
	replaceEC2 := hasConfigKey(rawConfiguration, "ec2Action")
	if replaceSNS || replaceEC2 {
		ec2ActionARN := ""
		if ec2Action := strings.TrimSpace(config.EC2Action); ec2Action != "" {
			if err := requireEC2ActionTarget(existing.Namespace, existing.Dimensions); err != nil {
				return PutMetricAlarmInput{}, err
			}

			ec2ActionARN = EC2AutomationARN(region, ec2Action)
		}

		input.AlarmActions = mergeAlarmActions(existing.AlarmActions, config.AlarmActions, ec2ActionARN, replaceSNS, replaceEC2)
	}

	// These transitions expose only SNS topics, so any other ARN on them is
	// preserved for the same reason it is on the ALARM transition.
	if hasConfigKey(rawConfiguration, "okActions") {
		input.OKActions = mergeAlarmActions(existing.OKActions, config.OKActions, "", true, false)
	}

	if hasConfigKey(rawConfiguration, "insufficientDataActions") {
		input.InsufficientDataActions = mergeAlarmActions(
			existing.InsufficientDataActions, config.InsufficientDataActions, "", true, false,
		)
	}

	return input, nil
}

func (c *UpdateAlarm) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateAlarm) HandleHook(_ core.ActionHookContext) error {
	return nil
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
