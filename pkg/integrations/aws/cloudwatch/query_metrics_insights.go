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

const (
	defaultLookbackMinutes         = 15
	defaultMaxDatapoints           = 1000
	maxAllowedCloudWatchDatapoints = 100800
	ScanByTimestampDescending      = "TimestampDescending"
	ScanByTimestampAscending       = "TimestampAscending"
	QueryMetricsInsightsEventType = "aws.cloudwatch.metricsInsights.query"
)

var AllScanByOptions = []configuration.FieldOption{
	{
		Label: "Timestamp Descending",
		Value: ScanByTimestampDescending,
	},
	{
		Label: "Timestamp Ascending",
		Value: ScanByTimestampAscending,
	},
}

type QueryMetricsInsights struct{}

type QueryMetricsInsightsConfiguration struct {
	Region          string `json:"region" mapstructure:"region"`
	Query           string `json:"query" mapstructure:"query"`
	LookbackMinutes int    `json:"lookbackMinutes" mapstructure:"lookbackMinutes"`
	MaxDatapoints   int    `json:"maxDatapoints" mapstructure:"maxDatapoints"`
	ScanBy          string `json:"scanBy" mapstructure:"scanBy"`
}

func (c *QueryMetricsInsights) Name() string {
	return "aws.cloudwatch.queryMetricsInsights"
}

func (c *QueryMetricsInsights) Label() string {
	return "CloudWatch • Query Metrics Insights"
}

func (c *QueryMetricsInsights) Description() string {
	return "Run a CloudWatch Metrics Insights query against AWS metrics"
}

func (c *QueryMetricsInsights) Documentation() string {
	return `The Query Metrics Insights component runs a CloudWatch Metrics Insights query using the GetMetricData API.

## Use Cases

- **Observability automation**: Query current metric trends during workflows
- **SLO checks**: Evaluate key service metrics before progressing a deployment
- **Incident response**: Pull grouped metric views to enrich notifications

## Configuration

- **Region**: AWS region where the metrics are stored
- **Metrics Insights Query**: SQL-like query in CloudWatch Metrics Insights syntax
- **Lookback Window (minutes)**: Relative time window ending at execution time
- **Max Datapoints**: Maximum datapoints returned by CloudWatch
- **Result Order**: Timestamp ascending or descending order

## Notes

- The component automatically sets the query window to now minus lookback through now
- It emits one payload containing query metadata and all metric result series`
}

func (c *QueryMetricsInsights) Icon() string {
	return "aws"
}

func (c *QueryMetricsInsights) Color() string {
	return "gray"
}

func (c *QueryMetricsInsights) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryMetricsInsights) Configuration() []configuration.Field {
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
			Name:        "query",
			Label:       "Metrics Insights Query",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "CloudWatch Metrics Insights query string (for example: SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId) GROUP BY InstanceId)",
			Placeholder: "SELECT AVG(CPUUtilization) FROM SCHEMA(\"AWS/EC2\", InstanceId) GROUP BY InstanceId",
		},
		{
			Name:        "lookbackMinutes",
			Label:       "Lookback Window (minutes)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultLookbackMinutes,
			Description: "How many minutes back from now to query",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := 1440; return &max }(),
				},
			},
		},
		{
			Name:        "maxDatapoints",
			Label:       "Max Datapoints",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultMaxDatapoints,
			Description: "Maximum datapoints CloudWatch should return",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 1; return &min }(),
					Max: func() *int { max := maxAllowedCloudWatchDatapoints; return &max }(),
				},
			},
		},
		{
			Name:     "scanBy",
			Label:    "Result Order",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  ScanByTimestampDescending,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: AllScanByOptions,
				},
			},
		},
	}
}

func (c *QueryMetricsInsights) Setup(ctx core.SetupContext) error {
	_, err := c.parseConfiguration(ctx.Configuration)
	return err
}

func (c *QueryMetricsInsights) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *QueryMetricsInsights) Execute(ctx core.ExecutionContext) error {
	config, err := c.parseConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	endTime := time.Now().UTC()
	startTime := endTime.Add(-time.Duration(config.LookbackMinutes) * time.Minute)

	client := NewClient(ctx.HTTP, credentials, config.Region)
	response, err := client.QueryMetricsInsights(QueryMetricsInsightsInput{
		Query:         config.Query,
		StartTime:     startTime,
		EndTime:       endTime,
		ScanBy:        config.ScanBy,
		MaxDatapoints: config.MaxDatapoints,
	})
	if err != nil {
		return fmt.Errorf("failed to execute metrics insights query: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		QueryMetricsInsightsEventType,
		[]any{
			map[string]any{
				"region":        config.Region,
				"query":         config.Query,
				"startTime":     startTime.Format(time.RFC3339),
				"endTime":       endTime.Format(time.RFC3339),
				"scanBy":        config.ScanBy,
				"maxDatapoints": config.MaxDatapoints,
				"requestId":     response.RequestID,
				"results":       response.Results,
				"messages":      response.Messages,
			},
		},
	)
}

func (c *QueryMetricsInsights) Actions() []core.Action {
	return []core.Action{}
}

func (c *QueryMetricsInsights) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *QueryMetricsInsights) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *QueryMetricsInsights) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *QueryMetricsInsights) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *QueryMetricsInsights) parseConfiguration(rawConfiguration any) (QueryMetricsInsightsConfiguration, error) {
	config := QueryMetricsInsightsConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Query = strings.TrimSpace(config.Query)
	config.ScanBy = strings.TrimSpace(config.ScanBy)

	if config.LookbackMinutes < 0 {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("lookback minutes must be greater than or equal to zero")
	}
	if config.MaxDatapoints < 0 {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("max datapoints must be greater than or equal to zero")
	}

	if config.LookbackMinutes == 0 {
		config.LookbackMinutes = defaultLookbackMinutes
	}
	if config.MaxDatapoints == 0 {
		config.MaxDatapoints = defaultMaxDatapoints
	}
	if config.ScanBy == "" {
		config.ScanBy = ScanByTimestampDescending
	}

	if config.Region == "" {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("region is required")
	}
	if config.Query == "" {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("metrics insights query is required")
	}
	if config.LookbackMinutes < 1 {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("lookback minutes must be greater than zero")
	}
	if config.MaxDatapoints < 1 {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("max datapoints must be greater than zero")
	}
	if config.MaxDatapoints > maxAllowedCloudWatchDatapoints {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("max datapoints must be less than or equal to %d", maxAllowedCloudWatchDatapoints)
	}
	if !isValidScanBy(config.ScanBy) {
		return QueryMetricsInsightsConfiguration{}, fmt.Errorf("invalid scan by value: %s", config.ScanBy)
	}

	return config, nil
}

func isValidScanBy(scanBy string) bool {
	switch scanBy {
	case ScanByTimestampAscending, ScanByTimestampDescending:
		return true
	default:
		return false
	}
}
