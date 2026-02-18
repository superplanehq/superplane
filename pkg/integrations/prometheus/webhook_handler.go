package prometheus

import "github.com/superplanehq/superplane/pkg/core"

type PrometheusWebhookHandler struct{}

func (h *PrometheusWebhookHandler) CompareConfig(a any, b any) (bool, error) {
	return true, nil
}

func (h *PrometheusWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *PrometheusWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *PrometheusWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
