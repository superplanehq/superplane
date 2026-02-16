package linear

import "github.com/superplanehq/superplane/pkg/core"

// LinearWebhookHandler implements core.WebhookHandler for Linear.
// Webhook registration is manual — the user creates the webhook in Linear's settings
// and points it at the URL shown in the trigger's setup panel.
type LinearWebhookHandler struct{}

func (h *LinearWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *LinearWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *LinearWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (h *LinearWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
