package dash0

import "github.com/superplanehq/superplane/pkg/core"

type Dash0WebhookHandler struct{}

// Setup is a no-op: the user manually copies the webhook URL into Dash0.
func (h *Dash0WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

// Cleanup is a no-op: nothing to clean up in Dash0.
func (h *Dash0WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

// CompareConfig always returns true because all OnNotification triggers share a
// single integration-level webhook with an empty configuration.
func (h *Dash0WebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

// Merge returns the current configuration unchanged.
func (h *Dash0WebhookHandler) Merge(current, requested any) (merged any, changed bool, err error) {
	return current, false, nil
}
