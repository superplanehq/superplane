package newrelic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ReportMetricPayloadType = "newrelic.metric"

type ReportMetric struct{}

type ReportMetricSpec struct {
	MetricName string         `json:"metricName" yaml:"metricName"`
	MetricType string         `json:"metricType" yaml:"metricType"`
	Value      float64        `json:"value" yaml:"value"`
	Timestamp  int64          `json:"timestamp" yaml:"timestamp"`
	IntervalMs int64          `json:"intervalMs" yaml:"intervalMs"`
	Attributes map[string]any `json:"attributes" yaml:"attributes"`
}

func (c *ReportMetric) Name() string {
	return "newrelic.reportMetric"
}

func (c *ReportMetric) Label() string {
	return "Report Metric"
}

func (c *ReportMetric) Description() string {
	return "Send custom metrics to New Relic"
}

func (c *ReportMetric) Documentation() string {
	return `The Report Metric component allows you to send custom metrics (Gauge, Count, Summary) to New Relic.

## Configuration

- **Metric Name**: The name of the metric (e.g., "server.cpu.usage")
- **Metric Type**: The type of metric (Gauge, Count, or Summary)
- **Value**: The numeric value of the metric
- **Timestamp**: Optional Unix timestamp (milliseconds). Defaults to now.
- **Interval (ms)**: Required for Count and Summary metrics. The duration of the measurement window in milliseconds.
- **Attributes**: Optional JSON object with additional attributes

## Output

Returns the sent metric payload.
`
}

func (c *ReportMetric) Icon() string {
	return "newrelic"
}

func (c *ReportMetric) Color() string {
	return "green"
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
			Description: "The name of the metric (e.g. server.cpu.usage)",
			Placeholder: "server.cpu.usage",
		},
		{
			Name:        "metricType",
			Label:       "Metric Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The type of metric",
			Default:     "gauge",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Gauge", Value: "gauge"},
						{Label: "Count", Value: "count"},
						{Label: "Summary", Value: "summary"},
					},
				},
			},
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "The numeric value of the metric",
			Placeholder: "0",
		},
		{
			Name:        "timestamp",
			Label:       "Timestamp (ms)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Optional Unix timestamp in milliseconds. Defaults to now.",
		},
		{
			Name:        "intervalMs",
			Label:       "Interval (ms)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Required and must be > 0 for count and summary metrics. Represents the duration of the measurement window in milliseconds.",
		},
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional additional attributes",
		},
	}
}

func (c *ReportMetric) Setup(ctx core.SetupContext) error {
	spec := ReportMetricSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.MetricName == "" {
		return fmt.Errorf("metricName is required")
	}

	if spec.MetricType == "" {
		return fmt.Errorf("metricType is required")
	}

	switch spec.MetricType {
	case string(MetricTypeGauge):
		// no interval requirement
	case string(MetricTypeCount), string(MetricTypeSummary):
		if spec.IntervalMs <= 0 {
			return fmt.Errorf("intervalMs is required and must be > 0 for count and summary metrics")
		}
	default:
		return fmt.Errorf("unsupported metricType %q", spec.MetricType)
	}

	return nil
}

func (c *ReportMetric) Execute(ctx core.ExecutionContext) error {
	spec := ReportMetricSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	timestamp := spec.Timestamp
	if timestamp == 0 {
		timestamp = time.Now().UnixMilli()
	}

	if (spec.MetricType == string(MetricTypeCount) || spec.MetricType == string(MetricTypeSummary)) && spec.IntervalMs <= 0 {
		return fmt.Errorf("intervalMs is required and must be > 0 for count and summary metrics")
	}

	metric := Metric{
		Name:       spec.MetricName,
		Type:       MetricType(spec.MetricType),
		Value:      spec.Value,
		Timestamp:  timestamp,
		IntervalMs: spec.IntervalMs,
		Attributes: spec.Attributes,
	}

	batch := []MetricBatch{
		{
			Metrics: []Metric{metric},
		},
	}

	if err := client.ReportMetric(context.Background(), batch); err != nil {
		return fmt.Errorf("failed to report metric: %v", err)
	}

    output := map[string]any{
        "name":      metric.Name,
        "value":     metric.Value,
        "type":      metric.Type,
        "timestamp": metric.Timestamp,
        "intervalMs": metric.IntervalMs,
        "status":    "202 Accepted",
    }

    return ctx.ExecutionState.Emit(
        core.DefaultOutputChannel.Name,
        ReportMetricPayloadType,
        []any{output},
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
	return 200, nil
}

func (c *ReportMetric) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ReportMetric) SampleOutput() any {
    return map[string]any{
        "name":      "server.cpu.usage",
        "value":     95.5,
        "type":      "gauge",
        "timestamp": 1707584119000,
        "status":    "202 Accepted",
    }
}