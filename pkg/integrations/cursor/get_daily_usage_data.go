package cursor

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"time"
)

const (
	GetDailyUsageDataPayloadType = "cursor.getDailyUsageData.result"
)

type GetDailyUsageData struct{}

type GetDailyUsageDataSpec struct {
	StartDate string `json:"startDate" mapstructure:"startDate"`
	EndDate   string `json:"endDate" mapstructure:"endDate"`
}

type GetDailyUsageDataOutput struct {
	Data   []map[string]any `json:"data"`
	Period map[string]any   `json:"period"`
}

func (c *GetDailyUsageData) Name() string {
	return "cursor.getDailyUsageData"
}

func (c *GetDailyUsageData) Label() string {
	return "Get Daily Usage Data"
}

func (c *GetDailyUsageData) Description() string {
	return "Fetches daily team usage metrics from Cursor's Admin API."
}

func (c *GetDailyUsageData) Documentation() string {
	return `The Get Daily Usage Data component fetches team usage metrics from Cursor's Admin API.

## Use Cases

- **Usage reporting**: Track team productivity and AI usage patterns
- **Cost tracking**: Monitor usage-based requests and subscription consumption
- **Analytics dashboards**: Build custom dashboards with Cursor usage data

## How It Works

1. Fetches usage data for the specified date range from Cursor's Admin API
2. Returns detailed metrics per user including lines added/deleted, requests, and model usage

## Configuration

- **Start Date**: Start of the date range (YYYY-MM-DD format, defaults to 7 days ago)
- **End Date**: End of the date range (YYYY-MM-DD format, defaults to today)

## Output

The output includes per-user daily metrics:
- Lines added/deleted (total and accepted)
- Tab completions shown/accepted
- Composer, chat, and agent requests
- Subscription vs usage-based request counts
- Most used model and file extensions

## Notes

- Requires a valid Cursor Admin API key configured in the integration
- Only returns data for active users`
}

func (c *GetDailyUsageData) Icon() string {
	return "bar-chart"
}

func (c *GetDailyUsageData) Color() string {
	return "#10B981"
}

func (c *GetDailyUsageData) ExampleOutput() map[string]any {
	return map[string]any{
		"data": []map[string]any{
			{
				"date":                     1710720000000,
				"isActive":                 true,
				"totalLinesAdded":          1543,
				"totalLinesDeleted":        892,
				"acceptedLinesAdded":       1102,
				"acceptedLinesDeleted":     645,
				"totalApplies":             87,
				"totalAccepts":             73,
				"totalRejects":             14,
				"totalTabsShown":           342,
				"totalTabsAccepted":        289,
				"composerRequests":         45,
				"chatRequests":             128,
				"agentRequests":            12,
				"subscriptionIncludedReqs": 180,
				"usageBasedReqs":           5,
				"mostUsedModel":            "gpt-4",
				"email":                    "developer@company.com",
			},
		},
		"period": map[string]any{
			"startDate": 1710720000000,
			"endDate":   1710892800000,
		},
	}
}

func (c *GetDailyUsageData) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDailyUsageData) Configuration() []configuration.Field {
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
	}
}

func (c *GetDailyUsageData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetDailyUsageData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetDailyUsageData) Execute(ctx core.ExecutionContext) error {
	spec := GetDailyUsageDataSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	now := time.Now().UTC()

	startOfToday := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, time.UTC,
	)

	endOfDay := time.Date(
		now.Year(), now.Month(), now.Day(),
		23, 59, 59, 0, time.UTC,
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
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, time.UTC)
	} else {
		endDate = endOfDay
	}

	if startDate.After(endDate) {
		return fmt.Errorf("start date must be before end date")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create cursor client: %w", err)
	}

	if client.AdminKey == "" {
		return fmt.Errorf("admin API key is not configured in the integration")
	}

	req := UsageRequest{
		StartDate: startDate.Unix() * 1000,
		EndDate:   endDate.Unix() * 1000,
	}

	ctx.Logger.Infof("Fetching Cursor usage data from %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	response, err := client.GetDailyUsage(req)
	if err != nil {
		return fmt.Errorf("failed to fetch usage data: %w", err)
	}

	output := GetDailyUsageDataOutput{
		Data: []map[string]any{},
		Period: map[string]any{
			"startDate": req.StartDate,
			"endDate":   req.EndDate,
		},
	}

	if data, ok := (*response)["data"].([]any); ok {
		for _, item := range data {
			if itemMap, ok := item.(map[string]any); ok {
				output.Data = append(output.Data, itemMap)
			}
		}
	}

	if period, ok := (*response)["period"].(map[string]any); ok {
		output.Period = period
	}

	ctx.Logger.Infof("Retrieved usage data for %d users", len(output.Data))

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetDailyUsageDataPayloadType, []any{output})

}

func (c *GetDailyUsageData) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetDailyUsageData) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetDailyUsageData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetDailyUsageData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetDailyUsageData) Cleanup(ctx core.SetupContext) error {
	return nil
}
