package honeycomb

import (
	"github.com/superplanehq/superplane/pkg/core"
)

type HoneycombWebhookHandler struct{}

func (h *HoneycombWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (h *HoneycombWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *HoneycombWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *HoneycombWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
