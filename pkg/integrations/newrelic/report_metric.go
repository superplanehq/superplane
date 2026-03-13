package newrelic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
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
	Interval   *int64         `json:"interval" mapstructure:"interval"`
}

const defaultIntervalMs = 60000

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
			Type:        configuration.FieldTypeString,
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
		{
			Name:        "interval",
			Label:       "Interval (ms)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultIntervalMs,
			Description: "Interval in milliseconds for count and summary metrics",
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

	if spec.MetricType == "summary" {
		valueMap, ok := spec.Value.(map[string]any)
		if !ok {
			return errors.New("summary metric value must be an object with keys: count, sum, min, max")
		}

		requiredKeys := []string{"count", "sum", "min", "max"}
		for _, key := range requiredKeys {
			if _, exists := valueMap[key]; !exists {
				return fmt.Errorf("summary metric value missing required key %q", key)
			}
		}
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
	if spec.Timestamp != nil && *spec.Timestamp > 0 {
		timestamp = *spec.Timestamp
	}

	numericValue, err := toNumericValue(spec.MetricType, spec.Value)
	if err != nil {
		return fmt.Errorf("invalid metric value: %v", err)
	}

	metric := map[string]any{
		"name":      spec.MetricName,
		"type":      spec.MetricType,
		"value":     numericValue,
		"timestamp": timestamp,
	}

	if spec.MetricType == "count" || spec.MetricType == "summary" {
		intervalMs := int64(defaultIntervalMs)
		if spec.Interval != nil {
			intervalMs = *spec.Interval
		}
		metric["interval.ms"] = intervalMs
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
			"value":      numericValue,
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

func (c *ReportMetric) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ReportMetric) Cleanup(ctx core.SetupContext) error {
	return nil
}

// toNumericValue coerces a metric value to the numeric type expected by the
// New Relic Metric API. For gauge and count metrics it returns a float64.
// For summary metrics it returns a map with float64 values.
func toNumericValue(metricType string, value any) (any, error) {
	if metricType == "summary" {
		return toSummaryValue(value)
	}

	return toFloat64(value)
}

func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to a number", n)
		}
		return f, nil
	case json.Number:
		return n.Float64()
	default:
		return 0, fmt.Errorf("unsupported value type %T", v)
	}
}

func toSummaryValue(value any) (map[string]any, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("summary value must be an object")
	}

	result := make(map[string]any, len(m))
	for _, key := range []string{"count", "sum", "min", "max"} {
		v, exists := m[key]
		if !exists {
			return nil, fmt.Errorf("summary value missing required key %q", key)
		}
		f, err := toFloat64(v)
		if err != nil {
			return nil, fmt.Errorf("summary key %q: %w", key, err)
		}
		result[key] = f
	}

	return result, nil
}
