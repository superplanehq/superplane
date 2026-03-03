package openrouter

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CurrentKeyDetailsPayloadType = "openrouter.currentKeyDetails"

type GetCurrentKeyDetails struct{}

type CurrentKeyDetailsPayload struct {
	Label              string   `json:"label"`
	Limit              *float64 `json:"limit,omitempty"`
	Usage              float64  `json:"usage"`
	UsageDaily         float64  `json:"usageDaily"`
	UsageWeekly        float64  `json:"usageWeekly"`
	UsageMonthly       float64  `json:"usageMonthly"`
	LimitRemaining     *float64 `json:"limitRemaining,omitempty"`
	LimitReset         *string  `json:"limitReset,omitempty"`
	IsFreeTier         bool     `json:"isFreeTier"`
	IsManagementKey    bool     `json:"isManagementKey"`
	IncludeByokInLimit bool     `json:"includeByokInLimit"`
	ExpiresAt          *string  `json:"expiresAt,omitempty"`
}

func (c *GetCurrentKeyDetails) Name() string {
	return "openrouter.getCurrentKeyDetails"
}

func (c *GetCurrentKeyDetails) Label() string {
	return "Get Current Key Details"
}

func (c *GetCurrentKeyDetails) Description() string {
	return "Get information on the API key associated with the current authentication (label, usage, limits)"
}

func (c *GetCurrentKeyDetails) Documentation() string {
	return `The Get Current Key Details component calls OpenRouter's GET /key API to return details for the authenticated API key.

## Configuration

No configuration. Uses the integration's API key.

## Output

Returns key details including:
- **label**: Human-readable label for the API key
- **limit**: Spending limit in USD (null if none)
- **usage**: Total OpenRouter credit usage in USD
- **usageDaily**, **usageWeekly**, **usageMonthly**: Usage for current UTC day/week/month
- **limitRemaining**: Remaining spending limit in USD (null if no limit)
- **limitReset**: Type of limit reset
- **isFreeTier**: Whether this is a free tier key
- **isManagementKey**: Whether this is a management key
- **includeByokInLimit**: Whether BYOK usage counts toward the limit
- **expiresAt**: ISO 8601 UTC expiration timestamp (null if no expiration)

## Use Cases

- **Key validation**: Verify the key in use and its label
- **Usage monitoring**: Check daily/weekly/monthly usage
- **Limit checks**: See remaining budget before running workflows
`
}

func (c *GetCurrentKeyDetails) Icon() string {
	return "key"
}

func (c *GetCurrentKeyDetails) Color() string {
	return "blue"
}

func (c *GetCurrentKeyDetails) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCurrentKeyDetails) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *GetCurrentKeyDetails) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCurrentKeyDetails) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	resp, err := client.GetCurrentKeyDetails()
	if err != nil {
		return err
	}

	d := resp.Data
	payload := CurrentKeyDetailsPayload{
		Label:              d.Label,
		Limit:              d.Limit,
		Usage:              d.Usage,
		UsageDaily:         d.UsageDaily,
		UsageWeekly:        d.UsageWeekly,
		UsageMonthly:       d.UsageMonthly,
		LimitRemaining:     d.LimitRemaining,
		LimitReset:         d.LimitReset,
		IsFreeTier:         d.IsFreeTier,
		IsManagementKey:    d.IsManagementKey,
		IncludeByokInLimit: d.IncludeByokInLimit,
		ExpiresAt:          d.ExpiresAt,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CurrentKeyDetailsPayloadType,
		[]any{payload},
	)
}

func (c *GetCurrentKeyDetails) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCurrentKeyDetails) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCurrentKeyDetails) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetCurrentKeyDetails) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetCurrentKeyDetails) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetCurrentKeyDetails) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCurrentKeyDetails) ExampleOutput() map[string]any {
	limit := 100.0
	remaining := 75.0
	return map[string]any{
		"type":               CurrentKeyDetailsPayloadType,
		"label":              "My API Key",
		"limit":              limit,
		"usage":              25.0,
		"usageDaily":         5.0,
		"usageWeekly":        20.0,
		"usageMonthly":       25.0,
		"limitRemaining":     remaining,
		"isFreeTier":         false,
		"isManagementKey":    false,
		"includeByokInLimit": false,
	}
}
