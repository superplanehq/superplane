package cursor

import "github.com/superplanehq/superplane/pkg/core"

type CursorWebhookHandler struct{}

func (h *CursorWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	// Cursor webhooks are configured per-agent at launch time. No external registration required.
	return nil, nil
}

func (h *CursorWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *CursorWebhookHandler) CompareConfig(a, b any) (bool, error) {
	// All Cursor nodes can share the same webhook record; executions are routed via agent_id KV.
	return true, nil
}
