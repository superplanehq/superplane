package digitalocean

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetDropletMetrics struct{}

type GetDropletMetricsSpec struct {
	Droplet        string `json:"droplet" mapstructure:"droplet"`
	LookbackPeriod string `json:"lookbackPeriod" mapstructure:"lookbackPeriod"`
}

var lookbackPeriodOptions = []configuration.FieldOption{
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
	{Label: "Last 14 days", Value: "14d"},
}

var lookbackDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

func (g *GetDropletMetrics) Name() string {
	return "digitalocean.getDropletMetrics"
}

func (g *GetDropletMetrics) Label() string {
	return "Get Droplet Metrics"
}

func (g *GetDropletMetrics) Description() string {
	return "Fetch CPU, memory, and network bandwidth metrics for a DigitalOcean Droplet"
}

func (g *GetDropletMetrics) Documentation() string {
	return `The Get Droplet Metrics component retrieves CPU usage, memory utilization, and network bandwidth metrics for a droplet over a specified lookback window.

> **Note:** Monitoring is only available for droplets that had monitoring enabled during creation. Droplets created without monitoring will not report metrics.

## Use Cases

- **Performance monitoring**: Sample current resource utilization before scaling decisions
- **Incident investigation**: Pull recent metrics when responding to an alert
- **Capacity planning**: Gather trend data to inform right-sizing of infrastructure
- **Automated scaling**: Use metric outputs to conditionally trigger resize or power operations

## Configuration

- **Droplet**: The droplet to fetch metrics for (required, supports expressions)
- **Lookback Period**: How far back to retrieve metrics — 1h, 6h, 24h, 7d, or 14d (required)

## Output

Returns a combined metrics payload with averaged values over the lookback window:
- **dropletId**: The ID of the queried droplet
- **start**: ISO 8601 timestamp of the start of the metrics window
- **end**: ISO 8601 timestamp of the end of the metrics window
- **lookbackPeriod**: The selected lookback period
- **avgCpuUsagePercent**: Average CPU usage percentage over the window
- **avgMemoryUsagePercent**: Average memory utilization percentage, computed from (total − available) / total × 100
- **avgPublicOutboundBandwidthMbps**: Average public outbound bandwidth in Mbps (as reported by the DigitalOcean API)
- **avgPublicInboundBandwidthMbps**: Average public inbound bandwidth in Mbps (as reported by the DigitalOcean API)

All metric values are rounded to two decimal places.

## Important Notes

- Metrics are only available for droplets with the DigitalOcean Monitoring Agent installed
- The Monitoring Agent is pre-installed on droplets using official DigitalOcean images created after 2018
- Data point resolution varies by window: shorter windows return finer-grained data`
}

func (g *GetDropletMetrics) Icon() string {
	return "chart-line"
}

func (g *GetDropletMetrics) Color() string {
	return "blue"
}

func (g *GetDropletMetrics) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDropletMetrics) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "droplet",
			Label:       "Droplet",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The droplet to fetch metrics for",
			Placeholder: "Select droplet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "droplet",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "lookbackPeriod",
			Label:       "Lookback Period",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "How far back to retrieve metrics data",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: lookbackPeriodOptions,
				},
			},
		},
	}
}

func (g *GetDropletMetrics) Setup(ctx core.SetupContext) error {
	spec := GetDropletMetricsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Droplet == "" {
		return errors.New("droplet is required")
	}

	if spec.LookbackPeriod == "" {
		return errors.New("lookbackPeriod is required")
	}

	if _, ok := lookbackDurations[spec.LookbackPeriod]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of 1h, 6h, 24h, 7d, 14d", spec.LookbackPeriod)
	}

	if err := resolveDropletMetadata(ctx, spec.Droplet); err != nil {
		return fmt.Errorf("error resolving droplet metadata: %v", err)
	}

	return nil
}

func (g *GetDropletMetrics) Execute(ctx core.ExecutionContext) error {
	spec := GetDropletMetricsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	duration, ok := lookbackDurations[spec.LookbackPeriod]
	if !ok {
		return fmt.Errorf("invalid lookbackPeriod %q", spec.LookbackPeriod)
	}

	now := time.Now().UTC()
	endTime := now
	startTime := now.Add(-duration)
	start := startTime.Unix()
	end := endTime.Unix()

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	cpu, err := client.GetDropletCPUMetrics(spec.Droplet, start, end)
	if err != nil {
		return fmt.Errorf("failed to get CPU metrics: %v", err)
	}

	memoryAvailable, err := client.GetDropletMemoryAvailableMetrics(spec.Droplet, start, end)
	if err != nil {
		return fmt.Errorf("failed to get memory available metrics: %v", err)
	}

	memoryTotal, err := client.GetDropletMemoryTotalMetrics(spec.Droplet, start, end)
	if err != nil {
		return fmt.Errorf("failed to get memory total metrics: %v", err)
	}

	outbound, err := client.GetDropletBandwidthMetrics(spec.Droplet, "public", "outbound", start, end)
	if err != nil {
		return fmt.Errorf("failed to get public outbound bandwidth metrics: %v", err)
	}

	inbound, err := client.GetDropletBandwidthMetrics(spec.Droplet, "public", "inbound", start, end)
	if err != nil {
		return fmt.Errorf("failed to get public inbound bandwidth metrics: %v", err)
	}

	cpuUsage := computeCPUUsagePercent(cpu)
	avgMemAvailable := averageGaugeMetrics(memoryAvailable)
	avgMemTotal := averageGaugeMetrics(memoryTotal)
	outboundMbps := averageGaugeMetrics(outbound)
	inboundMbps := averageGaugeMetrics(inbound)

	var memoryUsagePercent float64
	if avgMemTotal > 0 {
		memoryUsagePercent = roundTo((avgMemTotal-avgMemAvailable)/avgMemTotal*100, 2)
	}

	payload := map[string]any{
		"dropletId":                      spec.Droplet,
		"start":                          startTime.Format(time.RFC3339),
		"end":                            endTime.Format(time.RFC3339),
		"lookbackPeriod":                 spec.LookbackPeriod,
		"avgCpuUsagePercent":             roundTo(cpuUsage, 2),
		"avgMemoryUsagePercent":          memoryUsagePercent,
		"avgPublicOutboundBandwidthMbps": roundTo(outboundMbps, 2),
		"avgPublicInboundBandwidthMbps":  roundTo(inboundMbps, 2),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.droplet.metrics",
		[]any{payload},
	)
}

func parseMetricValue(v MetricsValue) (float64, bool) {
	if len(v) < 2 {
		return 0, false
	}

	switch val := v[1].(type) {
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

func computeCPUUsagePercent(resp *MetricsResponse) float64 {
	var totalDelta float64
	var idleDelta float64

	for _, result := range resp.Data.Result {
		if len(result.Values) < 2 {
			continue
		}

		first, ok := parseMetricValue(result.Values[0])
		if !ok {
			continue
		}

		last, ok := parseMetricValue(result.Values[len(result.Values)-1])
		if !ok {
			continue
		}

		delta := last - first
		if delta < 0 {
			// Counter reset — skip this series.
			continue
		}

		totalDelta += delta

		if result.Metric["mode"] == "idle" {
			idleDelta += delta
		}
	}

	if totalDelta == 0 {
		return 0
	}

	return (totalDelta - idleDelta) / totalDelta * 100
}

func averageGaugeMetrics(resp *MetricsResponse) float64 {
	var sum float64
	var count int

	for _, result := range resp.Data.Result {
		for _, v := range result.Values {
			f, ok := parseMetricValue(v)
			if !ok {
				continue
			}

			sum += f
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return sum / float64(count)
}

func roundTo(val float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(val*p) / p
}

func (g *GetDropletMetrics) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetDropletMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetDropletMetrics) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetDropletMetrics) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetDropletMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetDropletMetrics) Cleanup(ctx core.SetupContext) error {
	return nil
}
