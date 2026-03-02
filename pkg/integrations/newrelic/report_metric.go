package newrelic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ReportMetric struct{}

type ReportMetricSpec struct {
	MetricName string         `json:"metricName" mapstructure:"metricName"`
	MetricType string         `json:"metricType" mapstructure:"metricType"`
	Value      any            `json:"value" mapstructure:"value"`
	Attributes map[string]any `json:"attributes" mapstructure:"attributes"`
	Timestamp  *int64         `json:"timestamp" mapstructure:"timestamp"`
}

func (c *ReportMetric) Name() string {
	return "newrelic.reportMetric"
}

func (c *ReportMetric) Label() string {
	return "Report Metric"
}

func (c *ReportMetric) Description() string {
	return "Send custom metric data to New Relic"
}

func (c *ReportMetric) Icon() string {
	return "chart-bar"
}

func (c *ReportMetric) Color() string {
	return "gray"
}

func (c *ReportMetric) Documentation() string {
	return `The Report Metric component sends custom metric data to New Relic's Metric API.

## Use Cases

- **Deployment metrics**: Track deployment frequency and duration
- **Business metrics**: Report custom KPIs like revenue, signups, or conversion rates
- **Pipeline metrics**: Measure workflow execution times and success rates

## Configuration

- ` + "`metricName`" + `: The name of the metric (e.g., custom.deployment.count)
- ` + "`metricType`" + `: The type of metric (gauge, count, or summary)
- ` + "`value`" + `: The numeric value for the metric
- ` + "`attributes`" + `: Optional key-value labels for the metric
- ` + "`timestamp`" + `: Optional Unix epoch milliseconds (defaults to now)

## Outputs

The component emits a metric confirmation containing:
- ` + "`metricName`" + `: The name of the reported metric
- ` + "`metricType`" + `: The type of the metric
- ` + "`value`" + `: The reported value
- ` + "`timestamp`" + `: The timestamp used
`
}

func (c *ReportMetric) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ReportMetric) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "metricName",
			Label:       "Metric Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name of the metric (e.g., custom.deployment.count)",
			Placeholder: "custom.deployment.count",
		},
		{
			Name:     "metricType",
			Label:    "Metric Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "gauge",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Gauge", Value: "gauge"},
						{Label: "Count", Value: "count"},
						{Label: "Summary", Value: "summary"},
					},
				},
			},
			Description: "The type of metric to report",
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The numeric value for the metric",
		},
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional key-value attributes for the metric",
		},
		{
			Name:        "timestamp",
			Label:       "Timestamp",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Optional Unix epoch milliseconds (defaults to now)",
		},
	}
}

func (c *ReportMetric) Setup(ctx core.SetupContext) error {
	spec := ReportMetricSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.MetricName == "" {
		return errors.New("metricName is required")
	}

	if spec.MetricType == "" {
		return errors.New("metricType is required")
	}

	validMetricTypes := []string{"gauge", "count", "summary"}
	if !slices.Contains(validMetricTypes, spec.MetricType) {
		return fmt.Errorf("invalid metricType %q, must be one of: gauge, count, summary", spec.MetricType)
	}

	if spec.Value == nil {
		return errors.New("value is required")
	}

	return nil
}

func (c *ReportMetric) Execute(ctx core.ExecutionContext) error {
	spec := ReportMetricSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	timestamp := time.Now().UnixMilli()
	if spec.Timestamp != nil {
		timestamp = *spec.Timestamp
	}

	metric := map[string]any{
		"name":      spec.MetricName,
		"type":      spec.MetricType,
		"value":     spec.Value,
		"timestamp": timestamp,
	}

	if spec.MetricType == "count" || spec.MetricType == "summary" {
		metric["interval.ms"] = 60000
	}

	if spec.Attributes != nil && len(spec.Attributes) > 0 {
		metric["attributes"] = spec.Attributes
	}

	payload := []map[string]any{
		{
			"metrics": []map[string]any{metric},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %v", err)
	}

	_, err = client.ReportMetric(context.Background(), payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to report metric: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"newrelic.metric",
		[]any{map[string]any{
			"metricName": spec.MetricName,
			"metricType": spec.MetricType,
			"value":      spec.Value,
			"timestamp":  timestamp,
		}},
	)
}

func (c *ReportMetric) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ReportMetric) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ReportMetric) Actions() []core.Action {
	return []core.Action{}
}

func (c *ReportMetric) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ReportMetric) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ReportMetric) Cleanup(ctx core.SetupContext) error {
	return nil
}
