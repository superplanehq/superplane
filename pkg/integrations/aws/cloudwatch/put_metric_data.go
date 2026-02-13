package cloudwatch

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	storageResolutionStandard       = "60"
	storageResolutionHighResolution = "1"
	maxMetricDataPerRequest         = 1000
	maxDimensionsPerMetric          = 30
)

type PutMetricData struct{}

type PutMetricDataConfiguration struct {
	Region     string                             `json:"region" mapstructure:"region"`
	Namespace  string                             `json:"namespace" mapstructure:"namespace"`
	MetricData []PutMetricDatumConfigurationInput `json:"metricData" mapstructure:"metricData"`
}

type PutMetricDatumConfigurationInput struct {
	MetricName        string                             `json:"metricName" mapstructure:"metricName"`
	Value             *float64                           `json:"value" mapstructure:"value"`
	Unit              string                             `json:"unit" mapstructure:"unit"`
	Timestamp         string                             `json:"timestamp" mapstructure:"timestamp"`
	StorageResolution string                             `json:"storageResolution" mapstructure:"storageResolution"`
	Dimensions        []PutMetricDimensionConfiguration  `json:"dimensions" mapstructure:"dimensions"`
}

type PutMetricDimensionConfiguration struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type PutMetricDataOutput struct {
	RequestID   string   `json:"requestId"`
	Region      string   `json:"region"`
	Namespace   string   `json:"namespace"`
	MetricCount int      `json:"metricCount"`
	MetricNames []string `json:"metricNames"`
}

func (c *PutMetricData) Name() string {
	return "aws.cloudwatch.putMetricData"
}

func (c *PutMetricData) Label() string {
	return "CloudWatch • Put Metric Data"
}

func (c *PutMetricData) Description() string {
	return "Push custom metrics to AWS CloudWatch"
}

func (c *PutMetricData) Documentation() string {
	return `The Put Metric Data component publishes one or more custom metric data points to AWS CloudWatch.

## Use Cases

- **Application telemetry**: Publish request counts, latency, and error rates
- **Business KPIs**: Send domain metrics (orders, signups, revenue) to dashboards
- **Operational visibility**: Feed custom service health metrics into alarms and autoscaling

## Configuration

- **Region**: AWS region where the metric data will be published
- **Namespace**: CloudWatch namespace for your custom metrics (for example: `MyService/Production`)
- **Metric Data**: List of metric points to publish, each with:
  - **Metric Name** and **Value** (required)
  - Optional **Unit**, **Timestamp**, **Storage Resolution**, and **Dimensions**`
}

func (c *PutMetricData) Icon() string {
	return "aws"
}

func (c *PutMetricData) Color() string {
	return "gray"
}

func (c *PutMetricData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PutMetricData) Configuration() []configuration.Field {
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
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "MyService/Production",
			Description: "CloudWatch namespace to publish metrics to",
		},
		{
			Name:        "metricData",
			Label:       "Metric Data",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Metric data points to publish (up to 1000 per request)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Metric Data Point",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "metricName",
								Label:       "Metric Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Metric name",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeNumber,
								Required:    true,
								Description: "Metric value",
							},
							{
								Name:        "unit",
								Label:       "Unit",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Placeholder: "Count",
								Description: "Optional CloudWatch unit (for example: Count, Percent, Milliseconds)",
							},
							{
								Name:        "timestamp",
								Label:       "Timestamp",
								Type:        configuration.FieldTypeDateTime,
								Required:    false,
								Description: "Optional timestamp for the data point",
							},
							{
								Name:        "storageResolution",
								Label:       "Storage Resolution",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Description: "60 for standard metrics, 1 for high-resolution metrics",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Standard (60 seconds)", Value: storageResolutionStandard},
											{Label: "High Resolution (1 second)", Value: storageResolutionHighResolution},
										},
									},
								},
							},
							{
								Name:        "dimensions",
								Label:       "Dimensions",
								Type:        configuration.FieldTypeList,
								Required:    false,
								Description: "Optional dimensions as name/value pairs",
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel: "Dimension",
										ItemDefinition: &configuration.ListItemDefinition{
											Type: configuration.FieldTypeObject,
											Schema: []configuration.Field{
												{
													Name:     "name",
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
						},
					},
				},
			},
		},
	}
}

func (c *PutMetricData) Setup(ctx core.SetupContext) error {
	config := PutMetricDataConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return validatePutMetricDataConfiguration(config)
}

func (c *PutMetricData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PutMetricData) Execute(ctx core.ExecutionContext) error {
	config := PutMetricDataConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validatePutMetricDataConfiguration(config); err != nil {
		return err
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	metricData, metricNames, err := buildMetricData(config.MetricData)
	if err != nil {
		return err
	}

	client := NewClient(ctx.HTTP, credentials, strings.TrimSpace(config.Region))
	response, err := client.PutMetricData(strings.TrimSpace(config.Namespace), metricData)
	if err != nil {
		return fmt.Errorf("failed to put metric data: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.cloudwatch.metricData", []any{
		PutMetricDataOutput{
			RequestID:   response.RequestID,
			Region:      strings.TrimSpace(config.Region),
			Namespace:   strings.TrimSpace(config.Namespace),
			MetricCount: len(metricData),
			MetricNames: metricNames,
		},
	})
}

func (c *PutMetricData) Actions() []core.Action {
	return []core.Action{}
}

func (c *PutMetricData) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PutMetricData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *PutMetricData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PutMetricData) Cleanup(ctx core.SetupContext) error {
	return nil
}

func validatePutMetricDataConfiguration(config PutMetricDataConfiguration) error {
	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	namespace := strings.TrimSpace(config.Namespace)
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if len(config.MetricData) == 0 {
		return fmt.Errorf("metric data is required")
	}

	if len(config.MetricData) > maxMetricDataPerRequest {
		return fmt.Errorf("metric data exceeds limit of %d entries", maxMetricDataPerRequest)
	}

	for i, metric := range config.MetricData {
		if strings.TrimSpace(metric.MetricName) == "" {
			return fmt.Errorf("metricData[%d].metricName is required", i)
		}

		if metric.Value == nil {
			return fmt.Errorf("metricData[%d].value is required", i)
		}

		if strings.TrimSpace(metric.Timestamp) != "" {
			if _, err := parseMetricTimestamp(metric.Timestamp); err != nil {
				return fmt.Errorf("metricData[%d].timestamp must be a valid datetime", i)
			}
		}

		storageResolution := strings.TrimSpace(metric.StorageResolution)
		if storageResolution != "" &&
			storageResolution != storageResolutionStandard &&
			storageResolution != storageResolutionHighResolution {
			return fmt.Errorf(
				"metricData[%d].storageResolution must be %s or %s",
				i,
				storageResolutionHighResolution,
				storageResolutionStandard,
			)
		}

		if len(metric.Dimensions) > maxDimensionsPerMetric {
			return fmt.Errorf("metricData[%d].dimensions exceeds limit of %d entries", i, maxDimensionsPerMetric)
		}

		for j, dimension := range metric.Dimensions {
			if strings.TrimSpace(dimension.Name) == "" {
				return fmt.Errorf("metricData[%d].dimensions[%d].name is required", i, j)
			}
			if strings.TrimSpace(dimension.Value) == "" {
				return fmt.Errorf("metricData[%d].dimensions[%d].value is required", i, j)
			}
		}
	}

	return nil
}

func buildMetricData(metricInputs []PutMetricDatumConfigurationInput) ([]MetricDatum, []string, error) {
	metricData := make([]MetricDatum, 0, len(metricInputs))
	metricNames := make([]string, 0, len(metricInputs))

	for i, metric := range metricInputs {
		metricName := strings.TrimSpace(metric.MetricName)
		timestamp, err := parseMetricTimestamp(metric.Timestamp)
		if err != nil {
			return nil, nil, fmt.Errorf("metricData[%d].timestamp must be a valid datetime", i)
		}

		var storageResolution *int
		if strings.TrimSpace(metric.StorageResolution) != "" {
			value, err := strconv.Atoi(strings.TrimSpace(metric.StorageResolution))
			if err != nil {
				return nil, nil, fmt.Errorf("metricData[%d].storageResolution must be numeric", i)
			}
			storageResolution = &value
		}

		dimensions := make([]Dimension, 0, len(metric.Dimensions))
		for j, dimension := range metric.Dimensions {
			name := strings.TrimSpace(dimension.Name)
			value := strings.TrimSpace(dimension.Value)
			if name == "" {
				return nil, nil, fmt.Errorf("metricData[%d].dimensions[%d].name is required", i, j)
			}
			if value == "" {
				return nil, nil, fmt.Errorf("metricData[%d].dimensions[%d].value is required", i, j)
			}

			dimensions = append(dimensions, Dimension{Name: name, Value: value})
		}

		metricNames = append(metricNames, metricName)
		metricData = append(metricData, MetricDatum{
			MetricName:        metricName,
			Value:             *metric.Value,
			Unit:              strings.TrimSpace(metric.Unit),
			Timestamp:         timestamp,
			StorageResolution: storageResolution,
			Dimensions:        dimensions,
		})
	}

	return metricData, metricNames, nil
}

func parseMetricTimestamp(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}

	return nil, fmt.Errorf("invalid timestamp")
}
