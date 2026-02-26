package openrouter

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const RemainingCreditsPayloadType = "openrouter.remainingCredits"

type GetRemainingCredits struct{}

type RemainingCreditsPayload struct {
	TotalCredits float64 `json:"totalCredits"`
	TotalUsage   float64 `json:"totalUsage"`
	Remaining    float64 `json:"remaining"`
}

func (c *GetRemainingCredits) Name() string {
	return "openrouter.getRemainingCredits"
}

func (c *GetRemainingCredits) Label() string {
	return "Get Remaining Credits"
}

func (c *GetRemainingCredits) Description() string {
	return "Get total credits purchased and used for the authenticated OpenRouter account"
}

func (c *GetRemainingCredits) Documentation() string {
	return `The Get Remaining Credits component calls OpenRouter's credits API to return total credits purchased and used.

## Prerequisites

**Management API key required.** This endpoint only works with a [management key](https://openrouter.ai/docs/guides/overview/auth/management-api-keys). Standard API keys will receive 403 Forbidden.

## Configuration

No configuration. Uses the integration's API key (must be a management key).

## Output

Returns:
- **totalCredits**: Total credits purchased
- **totalUsage**: Total credits used
- **remaining**: totalCredits minus totalUsage

## Use Cases

- **Quota monitoring**: Check remaining credits before running expensive workflows
- **Billing alerts**: Notify when usage approaches limits
- **Dashboard reporting**: Display credit balance in internal tools
`
}

func (c *GetRemainingCredits) Icon() string {
	return "wallet"
}

func (c *GetRemainingCredits) Color() string {
	return "blue"
}

func (c *GetRemainingCredits) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRemainingCredits) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *GetRemainingCredits) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetRemainingCredits) Execute(ctx core.ExecutionContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	resp, err := client.GetRemainingCredits()
	if err != nil {
		return err
	}

	remaining := resp.Data.TotalCredits - resp.Data.TotalUsage
	payload := RemainingCreditsPayload{
		TotalCredits: resp.Data.TotalCredits,
		TotalUsage:   resp.Data.TotalUsage,
		Remaining:    remaining,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		RemainingCreditsPayloadType,
		[]any{payload},
	)
}

func (c *GetRemainingCredits) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRemainingCredits) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRemainingCredits) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRemainingCredits) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRemainingCredits) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetRemainingCredits) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetRemainingCredits) ExampleOutput() map[string]any {
	return map[string]any{
		"type":         RemainingCreditsPayloadType,
		"totalCredits": 100.0,
		"totalUsage":   25.5,
		"remaining":    74.5,
	}
}
