package sentry

import (
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type SentryWebhookHandler struct{}

func (h *SentryWebhookHandler) CompareConfig(a, b any) (bool, error) {
	// CompareConfig is only used to decide whether an existing webhook can be reused.
	// For Sentry, different trigger nodes can safely share a single webhook URL.
	// The actual event subscription is handled by Merge (which unions event sets).
	var cfgA, cfgB WebhookConfiguration
	if err := mapstructure.Decode(a, &cfgA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cfgB); err != nil {
		return false, err
	}

	return true, nil
}

func (h *SentryWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	// Return config as-is.
	return ctx.Webhook.GetConfiguration(), nil
}

func (h *SentryWebhookHandler) Merge(current, requested any) (any, bool, error) {
	var curCfg, reqCfg WebhookConfiguration
	if err := mapstructure.Decode(current, &curCfg); err != nil {
		return nil, false, err
	}
	if err := mapstructure.Decode(requested, &reqCfg); err != nil {
		return nil, false, err
	}

	seen := make(map[string]bool)
	mergedEvents := make([]string, 0, len(curCfg.Events)+len(reqCfg.Events))
	for _, e := range curCfg.Events {
		if seen[e] {
			continue
		}
		seen[e] = true
		mergedEvents = append(mergedEvents, e)
	}

	changed := false
	for _, e := range reqCfg.Events {
		if seen[e] {
			continue
		}
		seen[e] = true
		mergedEvents = append(mergedEvents, e)
		changed = true
	}

	merged := WebhookConfiguration{Events: mergedEvents}
	return merged, changed, nil
}

func (h *SentryWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
