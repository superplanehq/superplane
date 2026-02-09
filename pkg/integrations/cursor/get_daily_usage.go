package cursor

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const DailyUsagePayloadType = "cursor.usage.daily"

type GetDailyUsageData struct{}

type GetDailyUsageSpec struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

func (g *GetDailyUsageData) Name() string {
	return "cursor.getDailyUsageData"
}

func (g *GetDailyUsageData) Label() string {
	return "Get Daily Usage Data"
}

func (g *GetDailyUsageData) Description() string {
	return "Fetch Cursor team daily usage data (Admin API)"
}

func (g *GetDailyUsageData) Documentation() string {
	return `Fetches team daily usage data from the Cursor Admin API.

## Requirements

This component requires a Cursor Admin API key (team admin access). Some Cursor plans may not support this API.

## Configuration

- **Start Date**: Default ` + "`7d`" + ` (e.g., 7d, 30d, 2026-02-01, RFC3339)
- **End Date**: Default ` + "`today`" + ` (e.g., today, 2026-02-08)

## Output

Emits ` + "`cursor.usage.daily`" + ` with the response JSON from Cursor.`
}

func (g *GetDailyUsageData) Icon() string {
	return "bar-chart"
}

func (g *GetDailyUsageData) Color() string {
	return "gray"
}

func (g *GetDailyUsageData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetDailyUsageData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "startDate",
			Label:       "Start Date",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "7d",
			Placeholder: "7d",
			Description: "Relative (e.g., 7d) or absolute (YYYY-MM-DD/RFC3339)",
		},
		{
			Name:        "endDate",
			Label:       "End Date",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "today",
			Placeholder: "today",
			Description: "Relative (today/now) or absolute (YYYY-MM-DD/RFC3339)",
		},
	}
}

func (g *GetDailyUsageData) Setup(ctx core.SetupContext) error {
	return nil
}

func (g *GetDailyUsageData) Execute(ctx core.ExecutionContext) error {
	// Validate admin key exists.
	adminKey, err := ctx.Integration.GetConfig("adminApiKey")
	if err != nil || strings.TrimSpace(string(adminKey)) == "" {
		return fmt.Errorf("admin API key required. Cursor team admin access needed")
	}

	spec := GetDailyUsageSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	now := time.Now()
	startInput := spec.StartDate
	if strings.TrimSpace(startInput) == "" {
		startInput = "7d"
	}
	endInput := spec.EndDate
	if strings.TrimSpace(endInput) == "" {
		endInput = "today"
	}

	start, err := parseRelativeDate(startInput, now)
	if err != nil {
		return err
	}

	end, err := parseRelativeDate(endInput, now)
	if err != nil {
		return err
	}

	client, err := NewAdminClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	data, err := client.GetDailyUsageData(start, end)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DailyUsagePayloadType, []any{data})
}

func (g *GetDailyUsageData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetDailyUsageData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetDailyUsageData) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetDailyUsageData) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetDailyUsageData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (g *GetDailyUsageData) Cleanup(ctx core.SetupContext) error {
	return nil
}
