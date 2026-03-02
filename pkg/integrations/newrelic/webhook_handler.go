package newrelic

import "github.com/superplanehq/superplane/pkg/core"

type NewRelicWebhookHandler struct{}

func (h *NewRelicWebhookHandler) CompareConfig(a any, b any) (bool, error) {
	return true, nil
}

func (h *NewRelicWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	// Store the webhook URL in integration metadata so it is visible
	// in the integration configuration panel right away.
	if ctx.Integration != nil && ctx.Webhook != nil {
		ctx.Integration.SetMetadata(IntegrationMetadata{
			WebhookURL: ctx.Webhook.GetURL(),
		})
	}

	return nil, nil
}

func (h *NewRelicWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

// Merge always keeps the current config because all New Relic triggers share
// a single integration-level webhook with no trigger-specific configuration.
// CompareConfig returns true so all triggers route to the same webhook.
func (h *NewRelicWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
