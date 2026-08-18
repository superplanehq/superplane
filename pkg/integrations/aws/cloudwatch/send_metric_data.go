package cloudwatch

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const SendMetricDataPayloadType = "aws.cloudwatch.metricSent"

type SendMetricData struct{}

type SendMetricDataConfiguration struct {
	Region     string       `json:"region" mapstructure:"region"`
	Namespace  string       `json:"namespace" mapstructure:"namespace"`
	MetricName string       `json:"metricName" mapstructure:"metricName"`
	Value      float64      `json:"value" mapstructure:"value"`
	Unit       string       `json:"unit" mapstructure:"unit"`
	Dimensions []common.Tag `json:"dimensions" mapstructure:"dimensions"`
	Timestamp  string       `json:"timestamp" mapstructure:"timestamp"`
}

func (c *SendMetricData) Name() string {
	return "aws.cloudwatch.sendMetricData"
}

func (c *SendMetricData) Label() string {
	return "CloudWatch • Send Metric Data"
}

func (c *SendMetricData) Description() string {
	return "Publish a custom metric data point to CloudWatch from a workflow"
}

func (c *SendMetricData) Documentation() string {
	return `The Send Metric Data component publishes a single custom metric value to CloudWatch, creating the metric if it doesn't already exist.

## Use Cases

- **Workflow instrumentation**: Track business metrics (orders processed, jobs queued) alongside infrastructure metrics
- **Custom alarming**: Publish a metric, then alarm on it with the Create Alarm component
- **Cross-system correlation**: Emit values from external systems into CloudWatch for a unified view

## Configuration

- **Region**: AWS region to publish the metric to
- **Namespace**: Custom namespace for the metric. Must not start with ` + "`AWS/`" + `, which is reserved for AWS services
- **Metric Name**: Name of the metric
- **Value**: The value to publish
- **Unit** *(toggleable)*: Metric unit. Defaults to None
- **Dimensions** *(toggleable)*: Name/value pairs that further identify the metric
- **Timestamp** *(toggleable)*: When the value occurred; defaults to now. Must be within the last 2 weeks and no more than 2 hours in the future

## Output

- ` + "`namespace`" + `, ` + "`metricName`" + `, ` + "`value`" + `, ` + "`unit`" + `, ` + "`dimensions`" + `, ` + "`timestamp`" + ``
}

func (c *SendMetricData) Icon() string {
	return "aws"
}

func (c *SendMetricData) Color() string {
	return "gray"
}

func (c *SendMetricData) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendMetricData) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Custom namespace for the metric. Must not start with AWS/",
		},
		{
			Name:     "metricName",
			Label:    "Metric Name",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "value",
			Label:    "Value",
			Type:     configuration.FieldTypeNumber,
			Required: true,
		},
		{
			Name:      "unit",
			Label:     "Unit",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AlarmUnitOptions,
				},
			},
		},
		{
			Name:        "dimensions",
			Label:       "Dimensions",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Name/value pairs that further identify the metric",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Dimension",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "key",
								Label:    "Name",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "timestamp",
			Label:       "Timestamp",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Togglable:   true,
			Description: "Defaults to now",
		},
	}
}

func (c *SendMetricData) Setup(ctx core.SetupContext) error {
	config := SendMetricDataConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	_, err := validateSendMetricDataConfiguration(config)
	return err
}

func (c *SendMetricData) Execute(ctx core.ExecutionContext) error {
	config := SendMetricDataConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	timestamp, err := validateSendMetricDataConfiguration(config)
	if err != nil {
		return err
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	dimensions := make([]Dimension, 0, len(config.Dimensions))
	for _, tag := range config.Dimensions {
		dimensions = append(dimensions, Dimension{Name: tag.Key, Value: tag.Value})
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	err = client.PutMetricData(config.Namespace, MetricDatum{
		MetricName: config.MetricName,
		Value:      config.Value,
		Unit:       config.Unit,
		Timestamp:  timestamp,
		Dimensions: dimensions,
	})
	if err != nil {
		return fmt.Errorf("failed to send metric data: %w", err)
	}

	output := map[string]any{
		"namespace":  config.Namespace,
		"metricName": config.MetricName,
		"value":      config.Value,
		"unit":       config.Unit,
		"dimensions": dimensionsToMaps(dimensions),
		"timestamp":  timestamp.Format(time.RFC3339),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, SendMetricDataPayloadType, []any{output})
}

func (c *SendMetricData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendMetricData) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *SendMetricData) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *SendMetricData) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SendMetricData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendMetricData) Cleanup(ctx core.SetupContext) error {
	return nil
}

// validateSendMetricDataConfiguration validates the configuration and resolves the metric timestamp.
func validateSendMetricDataConfiguration(config SendMetricDataConfiguration) (time.Time, error) {
	if strings.TrimSpace(config.Region) == "" {
		return time.Time{}, fmt.Errorf("region is required")
	}

	namespace := strings.TrimSpace(config.Namespace)
	if namespace == "" {
		return time.Time{}, fmt.Errorf("namespace is required")
	}

	if strings.HasPrefix(namespace, "AWS/") {
		return time.Time{}, fmt.Errorf("namespace must not start with AWS/, which is reserved for AWS services")
	}

	if strings.TrimSpace(config.MetricName) == "" {
		return time.Time{}, fmt.Errorf("metric name is required")
	}

	timestampStr := strings.TrimSpace(config.Timestamp)
	if timestampStr == "" {
		return time.Now().UTC(), nil
	}

	return parseTimestamp(timestampStr)
}
