package cursor

import "github.com/superplanehq/superplane/pkg/core"

type CursorWebhookHandler struct{}

func (h *CursorWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (h *CursorWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
func (h *CursorWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *CursorWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
