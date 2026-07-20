package gcp

import "github.com/superplanehq/superplane/pkg/core"

// WebhookHandler is the integration-level webhook handler for GCP. It is a
// no-op: triggers that deliver through a SuperPlane node webhook (e.g.
// monitoring.onAlert, which creates its own Cloud Monitoring notification
// channel pointing at the node URL during Setup) still need the integration to
// expose a webhook handler so the webhook record can be provisioned and marked
// ready. The external-system wiring is owned by each trigger, so the handler
// itself has nothing to do.
type WebhookHandler struct{}

func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
