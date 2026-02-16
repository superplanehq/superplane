package sentry

import (
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct{}

type WebhookMetadata struct{}

type SentryWebhookHandler struct{}

func (h *SentryWebhookHandler) CompareConfig(a, b any) (bool, error) {
	// All Sentry webhooks for this integration use the same app
	// with the same events enabled, so configs are always compatible
	return true, nil
}

func (h *SentryWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *SentryWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	// Webhook is set up during the Sentry App creation in Sync()
	// No additional setup needed here
	return WebhookMetadata{}, nil
}

func (h *SentryWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	// Webhook cleanup happens in the main integration Cleanup()
	// No additional cleanup needed here
	return nil
}
