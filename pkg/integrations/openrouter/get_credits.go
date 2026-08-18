package openrouter

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetCreditsPayloadType = "openrouter.getCredits.result"

type GetCredits struct{}

type GetCreditsOutput struct {
	TotalCredits float64  `json:"totalCredits"`
	TotalUsage   float64  `json:"totalUsage"`
	Balance      float64  `json:"balance"`
	Key          KeyUsage `json:"key"`
}

// KeyUsage is the usage and limits of the API key the integration is
// configured with, as opposed to the account-wide credit totals.
type KeyUsage struct {
	Label          string   `json:"label"`
	Limit          *float64 `json:"limit"`
	LimitRemaining *float64 `json:"limitRemaining"`
	LimitReset     *string  `json:"limitReset"`
	Usage          float64  `json:"usage"`
	UsageDaily     float64  `json:"usageDaily"`
	UsageWeekly    float64  `json:"usageWeekly"`
	UsageMonthly   float64  `json:"usageMonthly"`
	IsFreeTier     bool     `json:"isFreeTier"`
}

func (c *GetCredits) Name() string {
	return "openrouter.getCredits"
}

func (c *GetCredits) Label() string {
	return "Get Credits"
}

func (c *GetCredits) Description() string {
	return "Fetches the OpenRouter credit balance and API key usage"
}

func (c *GetCredits) Documentation() string {
	return `The Get Credits component fetches the credit balance of your OpenRouter account together with the usage recorded against the configured API key.

## Use Cases

- **Budget alerts**: Trigger a notification when the remaining balance drops below a threshold
- **Spend reporting**: Track daily, weekly, and monthly usage on a schedule
- **Pre-flight checks**: Gate an expensive workflow on there being enough credit left

## How It Works

1. Reads the account credit totals purchased and consumed
2. Reads the usage and limits recorded against the configured API key
3. Emits both, along with the remaining balance

## Configuration

None. The component uses the API key configured on the integration.

## Output

The output includes:
- **totalCredits**: Credits purchased on the account
- **totalUsage**: Credits consumed on the account
- **balance**: Credits purchased minus credits consumed
- **key**: Usage recorded against the configured key — its label, all-time and daily/weekly/monthly usage, any credit limit and how much of it remains, and whether the account is on the free tier

## Notes

- Works with a normal API key; no provisioning key is required
- ` + "`key.limit`" + ` and ` + "`key.limitRemaining`" + ` are null when the key has no credit limit set
- Account credit totals cover the whole account, while the key figures cover only the configured key`
}

func (c *GetCredits) Icon() string {
	return "wallet"
}

func (c *GetCredits) Color() string {
	return "gray"
}

func (c *GetCredits) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCredits) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *GetCredits) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCredits) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	credits, err := client.GetCredits()
	if err != nil {
		return err
	}

	key, err := client.GetKey()
	if err != nil {
		return err
	}

	output := GetCreditsOutput{
		TotalCredits: credits.TotalCredits,
		TotalUsage:   credits.TotalUsage,
		Balance:      credits.TotalCredits - credits.TotalUsage,
		Key: KeyUsage{
			Label:          key.Label,
			Limit:          key.Limit,
			LimitRemaining: key.LimitRemaining,
			LimitReset:     key.LimitReset,
			Usage:          key.Usage,
			UsageDaily:     key.UsageDaily,
			UsageWeekly:    key.UsageWeekly,
			UsageMonthly:   key.UsageMonthly,
			IsFreeTier:     key.IsFreeTier,
		},
	}

	ctx.Logger.Infof("Retrieved OpenRouter credits: %.4f remaining", output.Balance)

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetCreditsPayloadType, []any{output})
}

func (c *GetCredits) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCredits) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetCredits) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCredits) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetCredits) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
