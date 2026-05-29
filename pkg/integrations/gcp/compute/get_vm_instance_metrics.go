package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const monitoringBaseURL = "https://monitoring.googleapis.com/v3"

type GetVMInstanceMetrics struct{}

type GetVMInstanceMetricsSpec struct {
	Instance       string `mapstructure:"instance"`
	LookbackPeriod string `mapstructure:"lookbackPeriod"`
}

var instanceLookbackOptions = []configuration.FieldOption{
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
	{Label: "Last 14 days", Value: "14d"},
}

var instanceLookbackDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

func (g *GetVMInstanceMetrics) Name() string {
	return "gcp.getVMInstanceMetrics"
}

func (g *GetVMInstanceMetrics) Label() string {
	return "Compute • Get VM Metrics"
}

func (g *GetVMInstanceMetrics) Description() string {
	return "Fetch CPU and network metrics for a Google Compute Engine VM instance"
}

func (g *GetVMInstanceMetrics) Documentation() string {
	return `The Get VM Metrics component retrieves CPU utilization and network throughput metrics for a Compute Engine VM instance from Cloud Monitoring over a specified lookback window.

## Use Cases

- **Performance monitoring**: Sample current resource utilization before scaling decisions
- **Incident investigation**: Pull recent metrics when responding to an alert
- **Capacity planning**: Gather trend data to inform right-sizing of infrastructure
- **Automated scaling**: Use metric outputs to conditionally trigger resize or power operations

## Configuration

- **VM Instance**: Pick from the list of VMs in your project, or pass an expression chained from an upstream node. The selection encodes both the zone and the instance name.
- **Lookback Period**: How far back to retrieve metrics — 1h, 6h, 24h, 7d, or 14d (required).

## Output

Returns an averaged metrics payload over the lookback window:
- **instanceId**: The numeric ID of the queried instance
- **name**, **zone**: Instance identity
- **start**, **end**: ISO 8601 timestamps of the metrics window
- **lookbackPeriod**: The selected lookback period
- **avgCpuUsagePercent**: Average CPU utilization percentage over the window
- **avgNetworkInboundBytesPerSec**: Average received network throughput in bytes/sec
- **avgNetworkOutboundBytesPerSec**: Average sent network throughput in bytes/sec

All metric values are rounded to two decimal places.

## Important Notes

- Requires the ` + "`roles/monitoring.viewer`" + ` IAM role on the integration's service account.
- Metrics are read from Cloud Monitoring (` + "`compute.googleapis.com`" + ` metrics), which are available for all Compute Engine instances by default.
- Data point resolution varies by window: shorter windows return finer-grained data.`
}

func (g *GetVMInstanceMetrics) Icon() string {
	return "chart-line"
}

func (g *GetVMInstanceMetrics) Color() string {
	return "blue"
}

func (g *GetVMInstanceMetrics) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetVMInstanceMetrics) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instance",
			Label:       "VM Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The VM instance to fetch metrics for. Lists every VM in your project across all zones.",
			Placeholder: "Select instance",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
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
					Options: instanceLookbackOptions,
				},
			},
		},
	}
}

func (g *GetVMInstanceMetrics) Setup(ctx core.SetupContext) error {
	spec := GetVMInstanceMetricsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.Instance) == "" {
		return errors.New("instance is required")
	}

	if spec.LookbackPeriod == "" {
		return errors.New("lookbackPeriod is required")
	}

	if _, ok := instanceLookbackDurations[spec.LookbackPeriod]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of 1h, 6h, 24h, 7d, 14d", spec.LookbackPeriod)
	}

	return resolveInstanceNodeMetadata(ctx, spec.Instance)
}

func (g *GetVMInstanceMetrics) Execute(ctx core.ExecutionContext) error {
	spec := GetVMInstanceMetricsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	duration, ok := instanceLookbackDurations[spec.LookbackPeriod]
	if !ok {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid lookbackPeriod %q", spec.LookbackPeriod))
	}

	urlProject, zone, instanceName, err := parseInstancePath(spec.Instance)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	if urlProject != "" && urlProject != project {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf(
			"instance belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			urlProject, project,
		))
	}

	callCtx := context.Background()

	// Cloud Monitoring filters by the numeric instance_id, so resolve it first.
	body, err := GetInstance(callCtx, client, project, zone, instanceName)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to read instance: %v", err))
	}
	instancePayload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to parse instance: %v", err))
	}
	instanceID, _ := instancePayload["instanceId"].(string)
	if instanceID == "" || instanceID == "0" {
		return ctx.ExecutionState.Fail("error", "could not resolve numeric instance ID for metrics query")
	}

	endTime := time.Now().UTC()
	startTime := endTime.Add(-duration)

	cpu, err := queryInstanceMetric(callCtx, client, project, instanceID,
		"compute.googleapis.com/instance/cpu/utilization", "ALIGN_MEAN", startTime, endTime)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get CPU metrics: %v", err))
	}

	inbound, err := queryInstanceMetric(callCtx, client, project, instanceID,
		"compute.googleapis.com/instance/network/received_bytes_count", "ALIGN_RATE", startTime, endTime)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get inbound network metrics: %v", err))
	}

	outbound, err := queryInstanceMetric(callCtx, client, project, instanceID,
		"compute.googleapis.com/instance/network/sent_bytes_count", "ALIGN_RATE", startTime, endTime)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get outbound network metrics: %v", err))
	}

	payload := map[string]any{
		"instanceId":                    instanceID,
		"name":                          instancePayload["name"],
		"zone":                          instancePayload["zone"],
		"start":                         startTime.Format(time.RFC3339),
		"end":                           endTime.Format(time.RFC3339),
		"lookbackPeriod":                spec.LookbackPeriod,
		"avgCpuUsagePercent":            roundTo(averageTimeSeries(cpu)*100, 2),
		"avgNetworkInboundBytesPerSec":  roundTo(averageTimeSeries(inbound), 2),
		"avgNetworkOutboundBytesPerSec": roundTo(averageTimeSeries(outbound), 2),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gcp.compute.vmInstance.metrics",
		[]any{payload},
	)
}

// timeSeriesResponse models the subset of the Cloud Monitoring timeSeries.list
// response that we read.
type timeSeriesResponse struct {
	TimeSeries []struct {
		Points []struct {
			Value struct {
				DoubleValue *float64 `json:"doubleValue"`
				Int64Value  *string  `json:"int64Value"`
			} `json:"value"`
		} `json:"points"`
	} `json:"timeSeries"`
}

// queryInstanceMetric calls Cloud Monitoring timeSeries.list for a single metric
// type, aligned with the given aligner over the window.
func queryInstanceMetric(ctx context.Context, client Client, project, instanceID, metricType, aligner string, start, end time.Time) (*timeSeriesResponse, error) {
	filter := fmt.Sprintf(`metric.type="%s" AND resource.labels.instance_id="%s"`, metricType, instanceID)

	q := url.Values{}
	q.Set("filter", filter)
	q.Set("interval.startTime", start.Format(time.RFC3339))
	q.Set("interval.endTime", end.Format(time.RFC3339))
	q.Set("aggregation.alignmentPeriod", "60s")
	q.Set("aggregation.perSeriesAligner", aligner)
	q.Set("view", "FULL")

	fullURL := fmt.Sprintf("%s/projects/%s/timeSeries?%s", monitoringBaseURL, project, q.Encode())

	body, err := client.GetURL(ctx, fullURL)
	if err != nil {
		return nil, err
	}

	var resp timeSeriesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse timeSeries response: %w", err)
	}

	return &resp, nil
}

// averageTimeSeries averages every point across all returned series.
func averageTimeSeries(resp *timeSeriesResponse) float64 {
	if resp == nil {
		return 0
	}

	var sum float64
	var count int
	for _, series := range resp.TimeSeries {
		for _, point := range series.Points {
			if point.Value.DoubleValue != nil {
				sum += *point.Value.DoubleValue
				count++
			} else if point.Value.Int64Value != nil {
				var f float64
				if _, err := fmt.Sscan(*point.Value.Int64Value, &f); err == nil {
					sum += f
					count++
				}
			}
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

func (g *GetVMInstanceMetrics) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetVMInstanceMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetVMInstanceMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetVMInstanceMetrics) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetVMInstanceMetrics) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetVMInstanceMetrics) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
