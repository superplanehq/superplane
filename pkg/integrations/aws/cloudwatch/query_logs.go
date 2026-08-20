package cloudwatch

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	queryLogsPollHook     = "pollQueryResults"
	queryLogsPollInterval = 3 * time.Second
	maxQueryLogsPollTries = 200

	queryStatusComplete  = "Complete"
	queryStatusScheduled = "Scheduled"
	queryStatusRunning   = "Running"

	QueryLogsPayloadType = "aws.cloudwatch.queryResult"

	// defaultQueryLogsLimit mirrors the "limit" field's UI default. A blank or
	// cleared field decodes to 0, which StartQuery would otherwise omit,
	// letting AWS apply its own much larger default instead of ours.
	defaultQueryLogsLimit = 100
)

var queryLookbackDurations = map[string]time.Duration{
	"15m": 15 * time.Minute,
	"1h":  time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
}

var queryLookbackOptions = []configuration.FieldOption{
	{Label: "Last 15 minutes", Value: "15m"},
	{Label: "Last 1 hour", Value: "1h"},
	{Label: "Last 6 hours", Value: "6h"},
	{Label: "Last 24 hours", Value: "24h"},
	{Label: "Last 7 days", Value: "7d"},
}

type QueryLogs struct{}

type QueryLogsConfiguration struct {
	Region         string   `json:"region" mapstructure:"region"`
	LogGroups      []string `json:"logGroups" mapstructure:"logGroups"`
	QueryString    string   `json:"queryString" mapstructure:"queryString"`
	LookbackPeriod string   `json:"lookbackPeriod" mapstructure:"lookbackPeriod"`
	Limit          int      `json:"limit" mapstructure:"limit"`
}

type QueryLogsExecutionMetadata struct {
	QueryID      string   `json:"queryId" mapstructure:"queryId"`
	Region       string   `json:"region" mapstructure:"region"`
	LogGroups    []string `json:"logGroups" mapstructure:"logGroups"`
	QueryString  string   `json:"queryString" mapstructure:"queryString"`
	Start        string   `json:"start" mapstructure:"start"`
	End          string   `json:"end" mapstructure:"end"`
	PollAttempts int      `json:"pollAttempts" mapstructure:"pollAttempts"`
}

func (c *QueryLogs) Name() string {
	return "aws.cloudwatch.queryLogs"
}

func (c *QueryLogs) Label() string {
	return "CloudWatch • Query Logs"
}

func (c *QueryLogs) Description() string {
	return "Run a CloudWatch Logs Insights query over one or more log groups and return structured rows"
}

func (c *QueryLogs) Documentation() string {
	return `The Query Logs component runs a CloudWatch Logs Insights query over one or more log groups and returns the matching rows.

## Use Cases

- **Incident investigation**: Pull recent error logs when responding to an alert
- **Auditing**: Search for specific events across log groups over a time window
- **Reporting**: Aggregate log data (e.g. ` + "`stats count(*) by field`" + `) for dashboards

## How It Works

1. Starts a Logs Insights query with ` + "`StartQuery`" + ` over the selected log groups and lookback window
2. Polls ` + "`GetQueryResults`" + ` until the query reaches a terminal state
3. Emits the matched rows once the query completes

## Configuration

- **Region**: AWS region of the log groups
- **Log Groups**: One or more CloudWatch log groups to query
- **Query String**: Logs Insights query, e.g. ` + "`fields @timestamp, @message | sort @timestamp desc`" + `. Leave off a trailing ` + "`limit`" + ` command so the Limit field below governs the row count
- **Lookback Period**: How far back to search, relative to now
- **Limit**: Maximum number of log events to return (up to 10000)

## Output

- ` + "`queryId`" + `, ` + "`status`" + `, ` + "`start`" + `, ` + "`end`" + `
- ` + "`rows`" + `: array of objects, one per matched log event, mapping field name to value
- ` + "`statistics`" + `: ` + "`bytesScanned`" + `, ` + "`recordsMatched`" + `, ` + "`recordsScanned`" + `

## Important Notes

- Queries time out after 60 minutes on the AWS side; this component gives up polling after about 10 minutes and fails the execution
- A single query can return at most 10000 rows`
}

func (c *QueryLogs) Icon() string {
	return "aws"
}

func (c *QueryLogs) Color() string {
	return "gray"
}

func (c *QueryLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *QueryLogs) Configuration() []configuration.Field {
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
			Name:        "logGroups",
			Label:       "Log Groups",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Log groups to query",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "region", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "cloudwatch.logGroup",
					Multi: true,
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
			Name:        "queryString",
			Label:       "Query String",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Default:     "fields @timestamp, @message | sort @timestamp desc",
			Description: "CloudWatch Logs Insights query. Leave off a trailing \"limit\" command so the Limit field below governs the row count instead",
		},
		{
			Name:        "lookbackPeriod",
			Label:       "Lookback Period",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "1h",
			Description: "How far back to query, relative to now",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: queryLookbackOptions,
				},
			},
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "100",
			Description: "Maximum number of log events to return",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(10000),
				},
			},
		},
	}
}

func (c *QueryLogs) Setup(ctx core.SetupContext) error {
	config := QueryLogsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return validateQueryLogsConfiguration(config)
}

func (c *QueryLogs) Execute(ctx core.ExecutionContext) error {
	config := QueryLogsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	if err := validateQueryLogsConfiguration(config); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get AWS credentials: %v", err))
	}

	duration := queryLookbackDurations[config.LookbackPeriod]
	endTime := time.Now().UTC()
	startTime := endTime.Add(-duration)

	limit := config.Limit
	if limit <= 0 {
		limit = defaultQueryLogsLimit
	}

	client := NewLogsClient(ctx.HTTP, creds, config.Region)
	queryID, err := client.StartQuery(StartQueryInput{
		LogGroupNames: config.LogGroups,
		QueryString:   config.QueryString,
		StartTime:     startTime,
		EndTime:       endTime,
		Limit:         limit,
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to start Logs Insights query: %v", err))
	}

	metadata := QueryLogsExecutionMetadata{
		QueryID:     queryID,
		Region:      config.Region,
		LogGroups:   config.LogGroups,
		QueryString: config.QueryString,
		Start:       startTime.Format(time.RFC3339),
		End:         endTime.Format(time.RFC3339),
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save execution metadata: %v", err))
	}

	return ctx.Requests.ScheduleActionCall(queryLogsPollHook, map[string]any{}, queryLogsPollInterval)
}

func (c *QueryLogs) Hooks() []core.Hook {
	return []core.Hook{
		{Name: queryLogsPollHook, Type: core.HookTypeInternal},
	}
}

func (c *QueryLogs) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != queryLogsPollHook {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("unknown action: %s", ctx.Name))
	}

	return c.pollQueryResults(ctx)
}

func (c *QueryLogs) pollQueryResults(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := QueryLogsExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode poll metadata: %v", err))
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get AWS credentials: %v", err))
	}

	client := NewLogsClient(ctx.HTTP, creds, metadata.Region)
	results, err := client.GetQueryResults(metadata.QueryID)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get query results: %v", err))
	}

	switch results.Status {
	case queryStatusComplete:
		return c.emitResults(ctx, metadata, results)

	case queryStatusScheduled, queryStatusRunning:
		return c.reschedulePoll(ctx, metadata)

	default:
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("Logs Insights query %s ended with status %s", metadata.QueryID, results.Status))
	}
}

func (c *QueryLogs) reschedulePoll(ctx core.ActionHookContext, metadata QueryLogsExecutionMetadata) error {
	metadata.PollAttempts++
	if metadata.PollAttempts >= maxQueryLogsPollTries {
		stopQuery(ctx.Logger, ctx.HTTP, ctx.Integration, metadata.Region, metadata.QueryID)
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("timed out waiting for Logs Insights query %s to complete after %d poll attempts",
			metadata.QueryID, metadata.PollAttempts))
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to save poll attempt count: %v", err))
	}

	return ctx.Requests.ScheduleActionCall(queryLogsPollHook, map[string]any{}, queryLogsPollInterval)
}

func (c *QueryLogs) emitResults(ctx core.ActionHookContext, metadata QueryLogsExecutionMetadata, results *QueryResults) error {
	payload := map[string]any{
		"queryId":     metadata.QueryID,
		"logGroups":   metadata.LogGroups,
		"queryString": metadata.QueryString,
		"start":       metadata.Start,
		"end":         metadata.End,
		"status":      results.Status,
		"rows":        rowsFromResults(results.Results),
		"statistics": map[string]any{
			"bytesScanned":   results.Statistics.BytesScanned,
			"recordsMatched": results.Statistics.RecordsMatched,
			"recordsScanned": results.Statistics.RecordsScanned,
		},
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, QueryLogsPayloadType, []any{payload})
}

func rowsFromResults(results [][]ResultField) []map[string]string {
	rows := make([]map[string]string, 0, len(results))
	for _, fields := range results {
		row := make(map[string]string, len(fields))
		for _, field := range fields {
			row[field.Field] = field.Value
		}
		rows = append(rows, row)
	}

	return rows
}

func validateQueryLogsConfiguration(config QueryLogsConfiguration) error {
	if strings.TrimSpace(config.Region) == "" {
		return fmt.Errorf("region is required")
	}

	if len(config.LogGroups) == 0 {
		return fmt.Errorf("at least one log group is required")
	}

	if strings.TrimSpace(config.QueryString) == "" {
		return fmt.Errorf("query string is required")
	}

	if _, ok := queryLookbackDurations[config.LookbackPeriod]; !ok {
		return fmt.Errorf("invalid lookbackPeriod %q: must be one of 15m, 1h, 6h, 24h, 7d", config.LookbackPeriod)
	}

	if config.Limit < 0 || config.Limit > 10000 {
		return fmt.Errorf("limit must be between 1 and 10000")
	}

	return nil
}

func (c *QueryLogs) Cancel(ctx core.ExecutionContext) error {
	metadata := QueryLogsExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil || metadata.QueryID == "" {
		return nil
	}

	stopQuery(ctx.Logger, ctx.HTTP, ctx.Integration, metadata.Region, metadata.QueryID)
	return nil
}

// stopQuery best-effort cancels a running Logs Insights query so it stops
// consuming the account's concurrent query quota once SuperPlane is no longer
// waiting on it. Failures are logged rather than returned: callers use this
// from paths (cancellation, poll timeout) that must complete either way.
func stopQuery(logger *log.Entry, http core.HTTPContext, integration core.IntegrationContext, region, queryID string) {
	creds, err := common.CredentialsFromInstallation(integration)
	if err != nil {
		logger.Warnf("failed to get AWS credentials to stop Logs Insights query %s: %v", queryID, err)
		return
	}

	if err := NewLogsClient(http, creds, region).StopQuery(queryID); err != nil {
		logger.Warnf("failed to stop Logs Insights query %s: %v", queryID, err)
		return
	}

	logger.Infof("stopped Logs Insights query %s", queryID)
}

func (c *QueryLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *QueryLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func intPtr(v int) *int {
	return &v
}
