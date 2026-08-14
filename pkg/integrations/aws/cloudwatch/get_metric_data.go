package cloudwatch

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const GetMetricDataPayloadType = "aws.cloudwatch.metricData"

var metricDataLookbackOptions = []configuration.FieldOption{
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
	{Label: "Last 14 days", Value: "14d"},
}

var metricDataLookbackDurations = map[string]time.Duration{
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

// metricDataLookbackResolution mirrors ec2.GetInstanceMetrics: 5-minute
// granularity for sub-day windows, hourly beyond that, to stay within
// CloudWatch's 1440 datapoints-per-request limit.
var metricDataLookbackResolution = map[string]int{
	"1h":  60,
	"6h":  300,
	"24h": 300,
	"7d":  3600,
	"14d": 3600,
}

const (
	metricDataServiceECS = "ecs"
	metricDataServiceALB = "alb"
	metricDataServiceRDS = "rds"
	metricDataServiceSQS = "sqs"
)

var metricDataNamespaces = map[string]string{
	metricDataServiceECS: "AWS/ECS",
	metricDataServiceALB: "AWS/ApplicationELB",
	metricDataServiceRDS: "AWS/RDS",
	metricDataServiceSQS: "AWS/SQS",
}

const (
	metricNameECSCPUUtilization      = "CPUUtilization"
	metricNameALBRequestCount        = "RequestCount"
	metricNameRDSDatabaseConnections = "DatabaseConnections"
	metricNameRDSFreeStorageSpace    = "FreeStorageSpace"
	metricNameRDSFreeableMemory      = "FreeableMemory"
	metricNameSQSMessagesVisible     = "ApproximateNumberOfMessagesVisible"
	metricNameSQSMessagesSent        = "NumberOfMessagesSent"
	metricNameSQSMessagesReceived    = "NumberOfMessagesReceived"
	metricNameSQSOldestMessageAge    = "ApproximateAgeOfOldestMessage"
)

// metricDataStatistics is the fixed aggregation CloudWatch applies for each
// curated metric, matching how the console graphs each one by default.
var metricDataStatistics = map[string]string{
	metricNameECSCPUUtilization:      StatisticAverage,
	metricNameALBRequestCount:        "Sum",
	metricNameRDSDatabaseConnections: StatisticAverage,
	metricNameRDSFreeStorageSpace:    StatisticAverage,
	metricNameRDSFreeableMemory:      StatisticAverage,
	metricNameSQSMessagesVisible:     StatisticAverage,
	metricNameSQSMessagesSent:        "Sum",
	metricNameSQSMessagesReceived:    "Sum",
	metricNameSQSOldestMessageAge:    StatisticAverage,
}

var rdsMetricOptions = []configuration.FieldOption{
	{Label: "CPU Utilization", Value: metricNameECSCPUUtilization},
	{Label: "Database Connections", Value: metricNameRDSDatabaseConnections},
	{Label: "Free Storage Space", Value: metricNameRDSFreeStorageSpace},
	{Label: "Freeable Memory", Value: metricNameRDSFreeableMemory},
}

var sqsMetricOptions = []configuration.FieldOption{
	{Label: "Messages Visible", Value: metricNameSQSMessagesVisible},
	{Label: "Messages Sent", Value: metricNameSQSMessagesSent},
	{Label: "Messages Received", Value: metricNameSQSMessagesReceived},
	{Label: "Oldest Message Age (seconds)", Value: metricNameSQSOldestMessageAge},
}

type GetMetricData struct{}

type GetMetricDataConfiguration struct {
	Region         string `json:"region" mapstructure:"region"`
	Service        string `json:"service" mapstructure:"service"`
	RDSMetric      string `json:"rdsMetric" mapstructure:"rdsMetric"`
	SQSMetric      string `json:"sqsMetric" mapstructure:"sqsMetric"`
	DimensionsECS  string `json:"dimensionsEcs" mapstructure:"dimensionsEcs"`
	DimensionsALB  string `json:"dimensionsAlb" mapstructure:"dimensionsAlb"`
	DimensionsRDS  string `json:"dimensionsRds" mapstructure:"dimensionsRds"`
	DimensionsSQS  string `json:"dimensionsSqs" mapstructure:"dimensionsSqs"`
	LookbackPeriod string `json:"lookbackPeriod" mapstructure:"lookbackPeriod"`
}

func (c *GetMetricData) Name() string {
	return "aws.cloudwatch.getMetricData"
}

func (c *GetMetricData) Label() string {
	return "CloudWatch • Get Metric Data"
}

func (c *GetMetricData) Description() string {
	return "Fetch ECS, ALB, RDS, or SQS metrics from CloudWatch over a lookback window"
}

func (c *GetMetricData) Documentation() string {
	return `The Get Metric Data component retrieves a CloudWatch metric for an ECS service, an Application Load Balancer, an RDS instance, or an SQS queue over a configurable lookback window, the same way ` + "`aws.ec2.getInstanceMetrics`" + ` does for EC2 instances.

## Use Cases

- **Performance monitoring**: Sample ECS CPU or ALB request volume before scaling decisions
- **Incident investigation**: Pull recent RDS or SQS metrics when responding to an alert
- **Capacity planning**: Gather trend data to inform right-sizing

## Configuration

- **Region**: AWS region the resource runs in
- **Service**: Which CloudWatch namespace to query — ECS Service CPU, ALB Request Count, RDS, or SQS
- **RDS Metric** / **SQS Metric**: Shown only for the RDS/SQS services; picks the specific metric to fetch
- **ECS Service / Load Balancer / RDS Instance / SQS Queue**: Dimension set to scope the metric to, discovered from what CloudWatch has actually recorded
- **Lookback Period**: How far back to retrieve metrics — 1h, 6h, 24h, 7d, or 14d

## Output

- ` + "`service`" + `, ` + "`namespace`" + `, ` + "`metricName`" + `, ` + "`statistic`" + `, ` + "`region`" + `, ` + "`lookbackPeriod`" + `, ` + "`start`" + `, ` + "`end`" + `
- ` + "`datapoints`" + `: array of ` + "`{timestamp, value}`" + ` objects, one per period in the window
- ` + "`aggregatedValue`" + `: the average (for Average-statistic metrics) or total (for Sum-statistic metrics) across the window; ` + "`null`" + ` when CloudWatch returns no datapoints

## Important Notes

- Dimensions are discovered from metrics CloudWatch has already recorded; a resource with no recent activity will have no dimensions to pick from
- All metric values are rounded to two decimal places`
}

func (c *GetMetricData) Icon() string {
	return "aws"
}

func (c *GetMetricData) Color() string {
	return "gray"
}

func (c *GetMetricData) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetMetricData) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "ECS Service • CPU Utilization", Value: metricDataServiceECS},
						{Label: "Application Load Balancer • Request Count", Value: metricDataServiceALB},
						{Label: "RDS Instance", Value: metricDataServiceRDS},
						{Label: "SQS Queue", Value: metricDataServiceSQS},
					},
				},
			},
		},
		{
			Name:     "rdsMetric",
			Label:    "RDS Metric",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "service", Values: []string{metricDataServiceRDS}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "service", Values: []string{metricDataServiceRDS}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: rdsMetricOptions,
				},
			},
		},
		{
			Name:     "sqsMetric",
			Label:    "SQS Metric",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "service", Values: []string{metricDataServiceSQS}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "service", Values: []string{metricDataServiceSQS}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: sqsMetricOptions,
				},
			},
		},
		{
			Name:        "dimensionsEcs",
			Label:       "ECS Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "ECS cluster/service to fetch CPU utilization for",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "service", Values: []string{metricDataServiceECS}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "service", Values: []string{metricDataServiceECS}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "cloudwatch.metricDimensions",
					Parameters: []configuration.ParameterRef{
						regionParameter(),
						{Name: "namespace", Value: strPtr(metricDataNamespaces[metricDataServiceECS])},
						{Name: "metricName", Value: strPtr(metricNameECSCPUUtilization)},
					},
				},
			},
		},
		{
			Name:        "dimensionsAlb",
			Label:       "Load Balancer",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Application Load Balancer to fetch request count for",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "service", Values: []string{metricDataServiceALB}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "service", Values: []string{metricDataServiceALB}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "cloudwatch.metricDimensions",
					Parameters: []configuration.ParameterRef{
						regionParameter(),
						{Name: "namespace", Value: strPtr(metricDataNamespaces[metricDataServiceALB])},
						{Name: "metricName", Value: strPtr(metricNameALBRequestCount)},
					},
				},
			},
		},
		{
			Name:        "dimensionsRds",
			Label:       "RDS Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "RDS instance to fetch the metric for",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "service", Values: []string{metricDataServiceRDS}},
				{Field: "rdsMetric", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "service", Values: []string{metricDataServiceRDS}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "cloudwatch.metricDimensions",
					Parameters: []configuration.ParameterRef{
						regionParameter(),
						{Name: "namespace", Value: strPtr(metricDataNamespaces[metricDataServiceRDS])},
						{Name: "metricName", ValueFrom: &configuration.ParameterValueFrom{Field: "rdsMetric"}},
					},
				},
			},
		},
		{
			Name:        "dimensionsSqs",
			Label:       "SQS Queue",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "SQS queue to fetch the metric for",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "service", Values: []string{metricDataServiceSQS}},
				{Field: "sqsMetric", Values: []string{"*"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "service", Values: []string{metricDataServiceSQS}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "cloudwatch.metricDimensions",
					Parameters: []configuration.ParameterRef{
						regionParameter(),
						{Name: "namespace", Value: strPtr(metricDataNamespaces[metricDataServiceSQS])},
						{Name: "metricName", ValueFrom: &configuration.ParameterValueFrom{Field: "sqsMetric"}},
					},
				},
			},
		},
		{
			Name:        "lookbackPeriod",
			Label:       "Lookback Period",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "1h",
			Description: "How far back to retrieve metrics data",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: metricDataLookbackOptions,
				},
			},
		},
	}
}

func (c *GetMetricData) Setup(ctx core.SetupContext) error {
	config := GetMetricDataConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return validateGetMetricDataConfiguration(config)
}

func (c *GetMetricData) Execute(ctx core.ExecutionContext) error {
	config := GetMetricDataConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	duration, ok := metricDataLookbackDurations[config.LookbackPeriod]
	if !ok {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("invalid lookbackPeriod %q: must be one of 1h, 6h, 24h, 7d, 14d", config.LookbackPeriod))
	}

	namespace, metricName, dimensions, err := resolveMetricDataQuery(config)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get AWS credentials: %v", err))
	}

	statistic := metricDataStatistics[metricName]
	endTime := time.Now().UTC()
	startTime := endTime.Add(-duration)

	client := NewClient(ctx.HTTP, creds, region)
	points, err := client.GetMetricStatistics(GetMetricStatisticsInput{
		Namespace:  namespace,
		MetricName: metricName,
		Dimensions: dimensions,
		StartTime:  startTime,
		EndTime:    endTime,
		Period:     metricDataLookbackResolution[config.LookbackPeriod],
		Statistic:  statistic,
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get metric data: %v", err))
	}

	payload := map[string]any{
		"service":         config.Service,
		"namespace":       namespace,
		"metricName":      metricName,
		"statistic":       statistic,
		"dimensions":      dimensionsToMaps(dimensions),
		"region":          region,
		"lookbackPeriod":  config.LookbackPeriod,
		"start":           startTime.Format(time.RFC3339),
		"end":             endTime.Format(time.RFC3339),
		"datapoints":      datapointsToMaps(points, statistic),
		"aggregatedValue": aggregateMetricValue(points, statistic),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetMetricDataPayloadType, []any{payload})
}

// validateGetMetricDataConfiguration checks the fields Execute needs before it
// resolves the service-specific namespace, metric, and dimensions.
func validateGetMetricDataConfiguration(config GetMetricDataConfiguration) error {
	if _, err := requireRegion(config.Region); err != nil {
		return err
	}

	if _, ok := metricDataLookbackDurations[config.LookbackPeriod]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of 1h, 6h, 24h, 7d, 14d", config.LookbackPeriod)
	}

	_, _, _, err := resolveMetricDataQuery(config)
	return err
}

// resolveMetricDataQuery picks the namespace, metric name, and dimensions for
// the configured service, validating that its dimension picker was filled in.
func resolveMetricDataQuery(config GetMetricDataConfiguration) (string, string, []Dimension, error) {
	switch config.Service {
	case metricDataServiceECS:
		dimensions, err := decodeRequiredDimensions(config.DimensionsECS, "ECS service")
		return metricDataNamespaces[metricDataServiceECS], metricNameECSCPUUtilization, dimensions, err

	case metricDataServiceALB:
		dimensions, err := decodeRequiredDimensions(config.DimensionsALB, "load balancer")
		return metricDataNamespaces[metricDataServiceALB], metricNameALBRequestCount, dimensions, err

	case metricDataServiceRDS:
		metricName, err := requireMetricName(config.RDSMetric)
		if err != nil {
			return "", "", nil, err
		}
		dimensions, err := decodeRequiredDimensions(config.DimensionsRDS, "RDS instance")
		return metricDataNamespaces[metricDataServiceRDS], metricName, dimensions, err

	case metricDataServiceSQS:
		metricName, err := requireMetricName(config.SQSMetric)
		if err != nil {
			return "", "", nil, err
		}
		dimensions, err := decodeRequiredDimensions(config.DimensionsSQS, "SQS queue")
		return metricDataNamespaces[metricDataServiceSQS], metricName, dimensions, err

	default:
		return "", "", nil, fmt.Errorf("invalid service %q: must be one of ecs, alb, rds, sqs", config.Service)
	}
}

func decodeRequiredDimensions(encoded, resourceLabel string) ([]Dimension, error) {
	if encoded == "" {
		return nil, fmt.Errorf("%s is required", resourceLabel)
	}

	dimensions, err := DecodeDimensions(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", resourceLabel, err)
	}

	return dimensions, nil
}

func dimensionsToMaps(dimensions []Dimension) []map[string]any {
	result := make([]map[string]any, 0, len(dimensions))
	for _, dimension := range dimensions {
		result = append(result, map[string]any{"name": dimension.Name, "value": dimension.Value})
	}

	return result
}

func datapointsToMaps(points []MetricDatapoint, statistic string) []map[string]any {
	result := make([]map[string]any, 0, len(points))
	for _, point := range points {
		result = append(result, map[string]any{
			"timestamp": point.Timestamp,
			"value":     roundMetricValue(metricDatapointValue(point, statistic), 2),
		})
	}

	return result
}

func aggregateMetricValue(points []MetricDatapoint, statistic string) any {
	if len(points) == 0 {
		return nil
	}

	if statistic == "Sum" {
		var total float64
		for _, point := range points {
			total += point.Sum
		}
		return roundMetricValue(total, 2)
	}

	var sum float64
	for _, point := range points {
		sum += point.Average
	}

	return roundMetricValue(sum/float64(len(points)), 2)
}

func metricDatapointValue(point MetricDatapoint, statistic string) float64 {
	if statistic == "Sum" {
		return point.Sum
	}

	return point.Average
}

func roundMetricValue(val float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(val*p) / p
}

func (c *GetMetricData) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetMetricData) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *GetMetricData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetMetricData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetMetricData) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetMetricData) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
