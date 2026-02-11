package cursor

import "github.com/superplanehq/superplane/pkg/core"

type CursorWebhookHandler struct{}

// CompareConfig checks if the webhook configuration has changed.
// Since we don't have project-level webhooks, this is always true.
func (h *CursorWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

// Merge handles merging new configuration with existing.
func (h *CursorWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// Setup is a no-op for Cursor because webhooks are set up "per-request"
// (Callback URL) inside the LaunchAgent logic, rather than "per-integration"
// (Subscription) like in Semaphore.
func (h *CursorWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

// Cleanup is a no-op as there are no global webhook resources to delete.
func (h *CursorWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
