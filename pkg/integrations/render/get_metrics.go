package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetMetricsPayloadType = "render.metrics"

type GetMetrics struct{}

type GetMetricsConfiguration struct {
	Resources         []string `json:"resources" mapstructure:"resources"`
	MetricTypes       []string `json:"metricTypes" mapstructure:"metricTypes"`
	StartTime         string   `json:"startTime" mapstructure:"startTime"`
	EndTime           string   `json:"endTime" mapstructure:"endTime"`
	ResolutionSeconds int      `json:"resolutionSeconds" mapstructure:"resolutionSeconds"`
	AggregationMethod string   `json:"aggregationMethod" mapstructure:"aggregationMethod"`
}

func (c *GetMetrics) Name() string { return "render.getMetrics" }

func (c *GetMetrics) Label() string { return "Get Metrics" }

func (c *GetMetrics) Description() string {
	return "Fetch Render CPU, memory, request, and connection metrics"
}

func (c *GetMetrics) Documentation() string {
	return `Fetch Render metrics and emit normalized latest, average, and maximum values plus raw series data.`
}

func (c *GetMetrics) Icon() string { return "activity" }

func (c *GetMetrics) Color() string { return "gray" }

func (c *GetMetrics) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetMetrics) Configuration() []configuration.Field {
	return []configuration.Field{
		stringListField("resources", "Resources", true, "Render service, Postgres, or Key Value resource IDs"),
		stringListField("metricTypes", "Metric Types", false, "Metrics to fetch: cpu, memory, http_requests, active_connections"),
		{Name: "startTime", Label: "Start Time", Type: configuration.FieldTypeString, Required: false, Description: "Optional RFC3339 start time"},
		{Name: "endTime", Label: "End Time", Type: configuration.FieldTypeString, Required: false, Description: "Optional RFC3339 end time"},
		{Name: "resolutionSeconds", Label: "Resolution Seconds", Type: configuration.FieldTypeNumber, Required: false, Default: 60},
		{Name: "aggregationMethod", Label: "Aggregation Method", Type: configuration.FieldTypeString, Required: false, Placeholder: "AVG, MAX, or MIN"},
	}
}

func decodeGetMetricsConfiguration(configuration any) (GetMetricsConfiguration, error) {
	spec := GetMetricsConfiguration{ResolutionSeconds: 60}
	if err := decodeActionConfiguration(configuration, &spec); err != nil {
		return GetMetricsConfiguration{}, err
	}
	spec.Resources = cleanStringList(spec.Resources)
	spec.MetricTypes = cleanStringList(spec.MetricTypes)
	spec.StartTime = strings.TrimSpace(spec.StartTime)
	spec.EndTime = strings.TrimSpace(spec.EndTime)
	spec.AggregationMethod = strings.ToUpper(strings.TrimSpace(spec.AggregationMethod))
	if len(spec.MetricTypes) == 0 {
		spec.MetricTypes = []string{"cpu", "memory"}
	}
	if len(spec.Resources) == 0 {
		return GetMetricsConfiguration{}, fmt.Errorf("at least one resource is required")
	}
	if spec.ResolutionSeconds < 30 {
		spec.ResolutionSeconds = 60
	}
	return spec, nil
}

func (c *GetMetrics) Setup(ctx core.SetupContext) error {
	_, err := decodeGetMetricsConfiguration(ctx.Configuration)
	return err
}

func (c *GetMetrics) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetMetricsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	query := MetricQuery{
		Resources:         spec.Resources,
		StartTime:         spec.StartTime,
		EndTime:           spec.EndTime,
		ResolutionSeconds: spec.ResolutionSeconds,
		AggregationMethod: spec.AggregationMethod,
	}

	metrics := map[string]any{}
	summaries := map[string]any{}
	for _, metricType := range spec.MetricTypes {
		series, err := client.GetMetric(metricType, query)
		if err != nil {
			return err
		}
		metrics[metricType] = metricSeriesPayloads(series)
		summaries[metricType] = metricSummary(series)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetMetricsPayloadType, []any{map[string]any{
		"resources": spec.Resources,
		"metrics":   metrics,
		"summaries": summaries,
	}})
}

func metricSeriesPayloads(series []MetricSeries) []map[string]any {
	result := make([]map[string]any, 0, len(series))
	for _, item := range series {
		values := make([]map[string]any, 0, len(item.Values))
		for _, value := range item.Values {
			values = append(values, map[string]any{"timestamp": value.Timestamp, "value": value.Value})
		}

		labels := make([]map[string]any, 0, len(item.Labels))
		for _, label := range item.Labels {
			labels = append(labels, map[string]any{"field": label.Field, "value": label.Value})
		}

		result = append(result, map[string]any{
			"labels": labels,
			"values": values,
			"unit":   item.Unit,
		})
	}
	return result
}

func metricSummary(series []MetricSeries) map[string]any {
	count := 0
	sum := 0.0
	max := 0.0
	latest := 0.0
	unit := ""

	for _, item := range series {
		if unit == "" {
			unit = item.Unit
		}
		for _, value := range item.Values {
			if count == 0 || value.Value > max {
				max = value.Value
			}
			latest = value.Value
			sum += value.Value
			count++
		}
	}

	avg := 0.0
	if count > 0 {
		avg = sum / float64(count)
	}

	return map[string]any{
		"latest": latest,
		"avg":    avg,
		"max":    max,
		"count":  count,
		"unit":   unit,
	}
}

func (c *GetMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *GetMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *GetMetrics) Cancel(ctx core.ExecutionContext) error      { return nil }
func (c *GetMetrics) Cleanup(ctx core.SetupContext) error         { return nil }
func (c *GetMetrics) Hooks() []core.Hook                          { return []core.Hook{} }
func (c *GetMetrics) HandleHook(ctx core.ActionHookContext) error { return nil }

func stringListField(name, label string, required bool, description string) configuration.Field {
	return configuration.Field{
		Name:        name,
		Label:       label,
		Type:        configuration.FieldTypeList,
		Required:    required,
		Description: description,
		TypeOptions: &configuration.TypeOptions{
			List: &configuration.ListTypeOptions{
				ItemLabel: label,
				ItemDefinition: &configuration.ListItemDefinition{
					Type: configuration.FieldTypeString,
				},
			},
		},
	}
}
