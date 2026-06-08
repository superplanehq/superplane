package ec2

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type GetInstanceMetrics struct{}

type GetInstanceMetricsConfiguration struct {
	Region         string `json:"region" mapstructure:"region"`
	InstanceID     string `json:"instance" mapstructure:"instance"`
	LookbackPeriod string `json:"lookbackPeriod" mapstructure:"lookbackPeriod"`
	IncludeMemory  bool   `json:"includeMemory" mapstructure:"includeMemory"`
}

type GetInstanceMetricsNodeMetadata struct {
	Region       string `json:"region" mapstructure:"region"`
	InstanceID   string `json:"instanceId" mapstructure:"instanceId"`
	InstanceName string `json:"instanceName" mapstructure:"instanceName"`
}

var metricsLookbackOptions = []configuration.FieldOption{
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
	{Label: "Last 14 days", Value: "14d"},
}

var metricsLookbackDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

// Resolution in seconds per lookback period. Uses 5-minute granularity for
// sub-day windows and hourly for longer ranges to stay within CloudWatch's
// 1440 datapoints-per-request limit.
var metricsLookbackResolution = map[string]int{
	"1h":  60,
	"6h":  300,
	"24h": 300,
	"7d":  3600,
	"14d": 3600,
}

func (c *GetInstanceMetrics) Name() string {
	return "aws.ec2.getInstanceMetrics"
}

func (c *GetInstanceMetrics) Label() string {
	return "EC2 • Get Instance Metrics"
}

func (c *GetInstanceMetrics) Description() string {
	return "Fetch CPU, network, and optional memory metrics for an EC2 instance via CloudWatch"
}

func (c *GetInstanceMetrics) Documentation() string {
	return `The Get Instance Metrics component retrieves CloudWatch metrics for an EC2 instance over a configurable lookback window.

> **Note:** CPU and network metrics are available for all instances. Memory metrics require the **CloudWatch Agent** to be installed and running on the instance.

## Use Cases

- **Performance monitoring**: Sample CPU and network utilisation before making scaling decisions
- **Incident investigation**: Pull recent metrics when responding to an alert
- **Capacity planning**: Gather trend data to inform right-sizing
- **Automated scaling**: Use metric outputs to conditionally trigger resize or power operations

## Configuration

- **Region**: AWS region where the instance runs
- **Instance**: EC2 instance to query metrics for
- **Lookback Period**: How far back to retrieve metrics — 1h, 6h, 24h, 7d, or 14d
- **Include Memory**: Fetch memory usage (requires the CloudWatch Agent on the instance)

## Output

Returns averaged and aggregated metrics over the lookback window:
- ` + "`instanceId`" + `, ` + "`region`" + `, ` + "`lookbackPeriod`" + `, ` + "`start`" + `, ` + "`end`" + `
- ` + "`avgCpuUsagePercent`" + `: Average CPU utilisation percentage (null when CloudWatch returns no datapoints for the window)
- ` + "`totalNetworkInBytes`" + `: Total inbound network bytes over the window
- ` + "`totalNetworkOutBytes`" + `: Total outbound network bytes over the window
- ` + "`avgNetworkInBytesPerSec`" + `: Average inbound bytes per second
- ` + "`avgNetworkOutBytesPerSec`" + `: Average outbound bytes per second
- ` + "`avgMemoryUsagePercent`" + `: Average memory utilisation (null when the CloudWatch Agent is unavailable, returns no datapoints, or the memory request fails)

## Important Notes

- CPU and network metrics use **basic monitoring** (5-minute resolution) by default; enable **detailed monitoring** on the instance for 1-minute resolution
- Memory metrics require the [CloudWatch Agent](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Install-CloudWatch-Agent.html) to be installed on the instance
- All metric values are rounded to two decimal places`
}

func (c *GetInstanceMetrics) Icon() string {
	return "aws"
}

func (c *GetInstanceMetrics) Color() string {
	return "gray"
}

func (c *GetInstanceMetrics) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetInstanceMetrics) Configuration() []configuration.Field {
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
			Name:        "instance",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "EC2 instance to fetch metrics for",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ec2.instance",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
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
					Options: metricsLookbackOptions,
				},
			},
		},
		{
			Name:        "includeMemory",
			Label:       "Include Memory",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     "false",
			Description: "Fetch memory usage percentage. Requires the CloudWatch Agent to be installed and running on the instance.",
		},
	}
}

func (c *GetInstanceMetrics) Setup(ctx core.SetupContext) error {
	config := GetInstanceMetricsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	if config.LookbackPeriod == "" {
		return errors.New("lookbackPeriod is required")
	}

	if _, ok := metricsLookbackDurations[config.LookbackPeriod]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of 1h, 6h, 24h, 7d, 14d", config.LookbackPeriod)
	}

	return ctx.Metadata.Set(GetInstanceMetricsNodeMetadata{
		Region:       region,
		InstanceID:   instanceID,
		InstanceName: resolveInstanceName(ctx, region, instanceID),
	})
}

func (c *GetInstanceMetrics) Execute(ctx core.ExecutionContext) error {
	config := GetInstanceMetricsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return err
	}

	instanceID, err := requireInstanceID(config.InstanceID)
	if err != nil {
		return err
	}

	duration, ok := metricsLookbackDurations[config.LookbackPeriod]
	if !ok {
		return fmt.Errorf("invalid lookbackPeriod %q", config.LookbackPeriod)
	}

	period := metricsLookbackResolution[config.LookbackPeriod]

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, region)

	now := time.Now().UTC()
	endTime := now
	startTime := now.Add(-duration)
	windowSeconds := int(duration.Seconds())

	metricInput := GetMetricStatisticsInput{
		InstanceID: instanceID,
		StartTime:  startTime,
		EndTime:    endTime,
		Period:     period,
	}

	type networkResult struct {
		points []CloudWatchDatapoint
		err    error
	}

	var (
		cpuPoints    []CloudWatchDatapoint
		netInResult  networkResult
		netOutResult networkResult
		cpuErr       error
		memPoints    []CloudWatchDatapoint
		memErr       error
		wg           sync.WaitGroup
		mu           sync.Mutex
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		input := metricInput
		input.Namespace = "AWS/EC2"
		input.MetricName = "CPUUtilization"
		input.Statistic = "Average"
		pts, e := client.GetMetricStatistics(input)
		mu.Lock()
		cpuPoints, cpuErr = pts, e
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		input := metricInput
		input.Namespace = "AWS/EC2"
		input.MetricName = "NetworkIn"
		input.Statistic = "Sum"
		pts, e := client.GetMetricStatistics(input)
		mu.Lock()
		netInResult = networkResult{pts, e}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		input := metricInput
		input.Namespace = "AWS/EC2"
		input.MetricName = "NetworkOut"
		input.Statistic = "Sum"
		pts, e := client.GetMetricStatistics(input)
		mu.Lock()
		netOutResult = networkResult{pts, e}
		mu.Unlock()
	}()

	if config.IncludeMemory {
		wg.Add(1)
		go func() {
			defer wg.Done()
			input := metricInput
			input.Namespace = "CWAgent"
			input.MetricName = "mem_used_percent"
			input.Statistic = "Average"
			pts, e := client.GetMetricStatistics(input)
			mu.Lock()
			memPoints, memErr = pts, e
			mu.Unlock()
		}()
	}

	wg.Wait()

	if cpuErr != nil {
		return fmt.Errorf("failed to get CPU metrics: %w", cpuErr)
	}
	if netInResult.err != nil {
		return fmt.Errorf("failed to get NetworkIn metrics: %w", netInResult.err)
	}
	if netOutResult.err != nil {
		return fmt.Errorf("failed to get NetworkOut metrics: %w", netOutResult.err)
	}

	totalNetIn := sumDatapoints(netInResult.points)
	totalNetOut := sumDatapoints(netOutResult.points)

	payload := map[string]any{
		"instanceId":               instanceID,
		"region":                   region,
		"lookbackPeriod":           config.LookbackPeriod,
		"start":                    startTime.Format(time.RFC3339),
		"end":                      endTime.Format(time.RFC3339),
		"totalNetworkInBytes":      roundMetric(totalNetIn, 2),
		"totalNetworkOutBytes":     roundMetric(totalNetOut, 2),
		"avgNetworkInBytesPerSec":  roundMetric(totalNetIn/float64(windowSeconds), 2),
		"avgNetworkOutBytesPerSec": roundMetric(totalNetOut/float64(windowSeconds), 2),
	}

	if len(cpuPoints) == 0 {
		payload["avgCpuUsagePercent"] = nil
	} else {
		payload["avgCpuUsagePercent"] = roundMetric(averageDatapoints(cpuPoints), 2)
	}

	if config.IncludeMemory {
		payload["avgMemoryUsagePercent"] = memoryUsagePercent(memErr, memPoints)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetInstanceMetricsPayloadType,
		[]any{payload},
	)
}

func averageDatapoints(points []CloudWatchDatapoint) float64 {
	if len(points) == 0 {
		return 0
	}

	var sum float64
	for _, p := range points {
		sum += p.Average
	}

	return sum / float64(len(points))
}

func memoryUsagePercent(memErr error, points []CloudWatchDatapoint) any {
	if memErr != nil || len(points) == 0 {
		return nil
	}

	return roundMetric(averageDatapoints(points), 2)
}

func sumDatapoints(points []CloudWatchDatapoint) float64 {
	var total float64
	for _, p := range points {
		total += p.Sum
	}

	return total
}

func roundMetric(val float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(val*p) / p
}

func (c *GetInstanceMetrics) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetInstanceMetrics) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *GetInstanceMetrics) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetInstanceMetrics) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetInstanceMetrics) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetInstanceMetrics) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
