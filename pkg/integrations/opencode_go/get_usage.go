package opencodego

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetUsagePayloadType = "opencodego.getUsage.result"

type GetUsage struct{}

// GetUsagePayload flattens the three limit windows so downstream nodes can
// read a threshold without digging through the response envelope.
type GetUsagePayload struct {
	RollingStatus   string     `json:"rollingStatus"`
	RollingPercent  float64    `json:"rollingPercent"`
	RollingResetsAt string     `json:"rollingResetsAt"`
	WeeklyStatus    string     `json:"weeklyStatus"`
	WeeklyPercent   float64    `json:"weeklyPercent"`
	WeeklyResetsAt  string     `json:"weeklyResetsAt"`
	MonthlyStatus   string     `json:"monthlyStatus"`
	MonthlyPercent  float64    `json:"monthlyPercent"`
	MonthlyResetsAt string     `json:"monthlyResetsAt"`
	Response        *PlanUsage `json:"response"`
}

func (c *GetUsage) Name() string {
	return "opencodego.getUsage"
}

func (c *GetUsage) Label() string {
	return "Get Usage"
}

func (c *GetUsage) Description() string {
	return "Fetches OpenCode Go plan limit windows and how much of each is used"
}

func (c *GetUsage) Documentation() string {
	return `The Get Usage component fetches how much of your OpenCode Go subscription's limit windows is used, straight from the gateway with the regular API key.

## Use Cases

- **Budget alerts**: Trigger a notification when the weekly or monthly window crosses a threshold
- **Pre-flight gating**: Hold an expensive workflow until the rolling window resets
- **Dashboards**: Track percent used across the rolling, weekly, and monthly windows

## How It Works

1. Reads the Go subscription's limit windows from the gateway
2. Emits status, percent used, and reset time for each window

## Configuration

None. It uses the regular API key set on the integration.

## Output

Each window (rolling, weekly, monthly) contributes three fields:

- **status**: "ok" while requests are served, or throttled once the window is exhausted
- **percent**: Share of the window's dollar cap already used
- **resetsAt**: When the window resets

The full response is also included under **response**.

## Notes

- The rolling window covers 5 hours ($12), the weekly window covers 7 days ($30), and the monthly window covers a month ($60). Limits are dollar values, so request counts depend on the model.
`
}

func (c *GetUsage) Icon() string {
	return "activity"
}

func (c *GetUsage) Color() string {
	return "gray"
}

func (c *GetUsage) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetUsage) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *GetUsage) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetUsage) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	ctx.Logger.Info("Fetching OpenCode Go usage windows")

	usage, err := client.GetPlanUsage()
	if err != nil {
		return err
	}

	payload := buildUsagePayload(usage)

	ctx.Logger.Infof("OpenCode Go usage: rolling %.0f%%, weekly %.0f%%, monthly %.0f%%",
		payload.RollingPercent, payload.WeeklyPercent, payload.MonthlyPercent)

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetUsagePayloadType, []any{payload})
}

// buildUsagePayload copies each window's fields to the top level.
func buildUsagePayload(usage *PlanUsage) *GetUsagePayload {
	return &GetUsagePayload{
		RollingStatus:   usage.Rolling.Status,
		RollingPercent:  usage.Rolling.Percent,
		RollingResetsAt: usage.Rolling.ResetsAt,
		WeeklyStatus:    usage.Weekly.Status,
		WeeklyPercent:   usage.Weekly.Percent,
		WeeklyResetsAt:  usage.Weekly.ResetsAt,
		MonthlyStatus:   usage.Monthly.Status,
		MonthlyPercent:  usage.Monthly.Percent,
		MonthlyResetsAt: usage.Monthly.ResetsAt,
		Response:        usage,
	}
}

func (c *GetUsage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetUsage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
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
