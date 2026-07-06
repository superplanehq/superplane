package openai

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	GetUsagePayloadType = "openai.getUsage.result"

	usageTypeCosts = "costs"

	// The Usage API returns at most 31 daily buckets per page; Costs allows 180.
	maxUsageBucketsPerPage = 31
	maxCostsBucketsPerPage = 180
)

// usageTypePaths maps the configured usage type to its org API path.
var usageTypePaths = map[string]string{
	"completions":               "/organization/usage/completions",
	"embeddings":                "/organization/usage/embeddings",
	"images":                    "/organization/usage/images",
	"moderations":               "/organization/usage/moderations",
	"audio_speeches":            "/organization/usage/audio_speeches",
	"audio_transcriptions":      "/organization/usage/audio_transcriptions",
	"vector_stores":             "/organization/usage/vector_stores",
	"code_interpreter_sessions": "/organization/usage/code_interpreter_sessions",
	usageTypeCosts:              "/organization/costs",
}

var usageTypeOptions = []configuration.FieldOption{
	{Label: "Completions", Value: "completions"},
	{Label: "Embeddings", Value: "embeddings"},
	{Label: "Images", Value: "images"},
	{Label: "Moderations", Value: "moderations"},
	{Label: "Audio Speeches", Value: "audio_speeches"},
	{Label: "Audio Transcriptions", Value: "audio_transcriptions"},
	{Label: "Vector Stores", Value: "vector_stores"},
	{Label: "Code Interpreter Sessions", Value: "code_interpreter_sessions"},
	{Label: "Costs", Value: usageTypeCosts},
}

var groupByOptions = []configuration.FieldOption{
	{Label: "None", Value: "none"},
	{Label: "Model", Value: "model"},
	{Label: "Project", Value: "project_id"},
	{Label: "Line Item (Costs only)", Value: "line_item"},
}

type GetUsage struct{}

type GetUsageSpec struct {
	StartDate string `json:"startDate" mapstructure:"startDate"`
	EndDate   string `json:"endDate" mapstructure:"endDate"`
	UsageType string `json:"usageType" mapstructure:"usageType"`
	GroupBy   string `json:"groupBy" mapstructure:"groupBy"`
}

type GetUsageOutput struct {
	Data      []map[string]any `json:"data"`
	Period    map[string]any   `json:"period"`
	UsageType string           `json:"usageType"`
}

func (c *GetUsage) Name() string {
	return "openai.getUsage"
}

func (c *GetUsage) Label() string {
	return "Get Usage Data"
}

func (c *GetUsage) Description() string {
	return "Fetches organization usage metrics from OpenAI's Usage API."
}

func (c *GetUsage) Documentation() string {
	return `The Get Usage Data component fetches organization usage metrics from OpenAI's Usage and Costs APIs.

## Use Cases

- **Usage reporting**: Track token consumption and request counts across models
- **Cost tracking**: Monitor daily spend with the Costs usage type
- **Analytics dashboards**: Build custom dashboards with OpenAI usage data

## How It Works

1. Fetches usage data for the specified date range from OpenAI's organization Usage API
2. Returns daily buckets of metrics, optionally grouped by model or project

## Configuration

- **Start Date**: Start of the date range (YYYY-MM-DD format, defaults to 7 days ago)
- **End Date**: End of the date range (YYYY-MM-DD format, defaults to today)
- **Usage Type**: The usage category to fetch (completions, embeddings, images, audio, costs, ...)
- **Group By**: (Optional) Group results by model or project. Line item grouping is only available for costs; model grouping is not available for costs, vector stores, or code interpreter sessions.

## Output

The output includes daily usage buckets. Each bucket carries its time range and results with metrics that depend on the usage type, e.g. for completions:
- Input/output/cached token counts
- Number of model requests
- Model and project identifiers (when grouped)

For costs, each result carries the amount value and currency per line item.

## Notes

- Requires an organization admin API key (sk-admin-...) configured in the integration
- Admin keys are created by organization owners at platform.openai.com/settings/organization/admin-keys
- Usage data is always fetched from the OpenAI platform API; a custom Base URL configured for model endpoints does not apply`
}

func (c *GetUsage) Icon() string {
	return "bar-chart"
}

func (c *GetUsage) Color() string {
	return "gray"
}

func (c *GetUsage) ExampleOutput() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"data": []map[string]any{
				{
					"object":     "bucket",
					"start_time": 1730419200,
					"end_time":   1730505600,
					"results": []map[string]any{
						{
							"object":              "organization.usage.completions.result",
							"input_tokens":        1000,
							"output_tokens":       500,
							"input_cached_tokens": 800,
							"num_model_requests":  5,
							"model":               "gpt-5.2",
						},
					},
				},
			},
			"period": map[string]any{
				"startDate": "2026-06-26",
				"endDate":   "2026-07-03",
			},
			"usageType": "completions",
		},
		"timestamp": "2026-07-03T12:00:00.000000000Z",
		"type":      GetUsagePayloadType,
	}
}

func (c *GetUsage) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetUsage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "startDate",
			Label:       "Start Date",
			Type:        configuration.FieldTypeString,
			Description: "YYYY-MM-DD (Defaults to 7 days ago)",
			Required:    false,
		},
		{
			Name:        "endDate",
			Label:       "End Date",
			Type:        configuration.FieldTypeString,
			Description: "YYYY-MM-DD (Defaults to today)",
			Required:    false,
		},
		{
			Name:        "usageType",
			Label:       "Usage Type",
			Type:        configuration.FieldTypeSelect,
			Description: "The usage category to fetch",
			Required:    false,
			Default:     "completions",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: usageTypeOptions,
				},
			},
		},
		{
			Name:        "groupBy",
			Label:       "Group By",
			Type:        configuration.FieldTypeSelect,
			Description: "Group results by model or project",
			Required:    false,
			Default:     "none",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: groupByOptions,
				},
			},
		},
	}
}

func (c *GetUsage) Setup(ctx core.SetupContext) error {
	spec := GetUsageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	usageType := spec.UsageType
	if usageType == "" {
		usageType = "completions"
	}

	if _, ok := usageTypePaths[usageType]; !ok {
		return fmt.Errorf("invalid usage type: %s", usageType)
	}

	return validateGroupBy(usageType, spec.GroupBy)
}

// validateGroupBy rejects group_by dimensions the Usage API does not support
// for the given usage type, so misconfigurations fail at config time.
func validateGroupBy(usageType, groupBy string) error {
	switch groupBy {
	case "", "none", "project_id":
		return nil
	case "line_item":
		if usageType != usageTypeCosts {
			return fmt.Errorf("line item grouping is only available for costs")
		}
		return nil
	case "model":
		switch usageType {
		case usageTypeCosts, "vector_stores", "code_interpreter_sessions":
			return fmt.Errorf("model grouping is not available for %s", usageType)
		}
		return nil
	default:
		return fmt.Errorf("invalid group by: %s", groupBy)
	}
}

func (c *GetUsage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetUsage) Execute(ctx core.ExecutionContext) error {
	spec := GetUsageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	usageType := spec.UsageType
	if usageType == "" {
		usageType = "completions"
	}

	path, ok := usageTypePaths[usageType]
	if !ok {
		return fmt.Errorf("invalid usage type: %s", usageType)
	}

	if err := validateGroupBy(usageType, spec.GroupBy); err != nil {
		return err
	}

	now := time.Now().UTC()

	startOfToday := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, time.UTC,
	)

	// Default: 7 days ago at 00:00:00 UTC
	startOfWeek := startOfToday.AddDate(0, 0, -7)

	var startDate, endDate time.Time
	var err error

	if spec.StartDate != "" {
		startDate, err = time.Parse("2006-01-02", spec.StartDate)
		if err != nil {
			return fmt.Errorf("invalid start date format (expected YYYY-MM-DD): %w", err)
		}
	} else {
		startDate = startOfWeek
	}

	if spec.EndDate != "" {
		endDate, err = time.Parse("2006-01-02", spec.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end date format (expected YYYY-MM-DD): %w", err)
		}
	} else {
		endDate = startOfToday
	}

	if startDate.After(endDate) {
		return fmt.Errorf("start date must be before end date")
	}

	// The Usage API treats end_time as an exclusive bound, so use midnight after
	// the end date to fully include the last day's bucket.
	endExclusive := endDate.AddDate(0, 0, 1)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create openai client: %w", err)
	}

	if client.AdminKey == "" {
		return fmt.Errorf("admin API key is not configured in the integration")
	}

	params := url.Values{}
	params.Set("start_time", strconv.FormatInt(startDate.Unix(), 10))
	params.Set("end_time", strconv.FormatInt(endExclusive.Unix(), 10))
	params.Set("bucket_width", "1d")
	params.Set("limit", strconv.Itoa(bucketLimit(usageType, startDate, endExclusive)))
	if spec.GroupBy != "" && spec.GroupBy != "none" {
		params.Set("group_by", spec.GroupBy)
	}

	ctx.Logger.Infof("Fetching OpenAI %s usage from %s to %s", usageType, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	buckets, err := client.GetUsage(path, params)
	if err != nil {
		return fmt.Errorf("failed to fetch usage data: %w", err)
	}

	output := GetUsageOutput{
		Data: make([]map[string]any, 0, len(buckets)),
		Period: map[string]any{
			"startDate": startDate.Format("2006-01-02"),
			"endDate":   endDate.Format("2006-01-02"),
		},
		UsageType: usageType,
	}

	for _, bucket := range buckets {
		output.Data = append(output.Data, map[string]any{
			"object":     bucket.Object,
			"start_time": bucket.StartTime,
			"end_time":   bucket.EndTime,
			"results":    bucket.Results,
		})
	}

	ctx.Logger.Infof("Retrieved %d usage buckets", len(output.Data))

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetUsagePayloadType, []any{output})
}

// bucketLimit returns the number of daily buckets to request per page, capped
// at the API maximum for the usage type. Both bounds are midnight-aligned, with
// endExclusive one day past the last included day.
func bucketLimit(usageType string, startDate, endExclusive time.Time) int {
	days := int(endExclusive.Sub(startDate).Hours() / 24)

	limit := maxUsageBucketsPerPage
	if usageType == usageTypeCosts {
		limit = maxCostsBucketsPerPage
	}

	if days < 1 {
		return 1
	}
	if days > limit {
		return limit
	}
	return days
}

func (c *GetUsage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetUsage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetUsage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetUsage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetUsage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
