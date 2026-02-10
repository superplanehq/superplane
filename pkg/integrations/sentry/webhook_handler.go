package sentry

import (
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type SentryWebhookHandler struct{}

func (h *SentryWebhookHandler) CompareConfig(a, b any) (bool, error) {
	var cfgA, cfgB WebhookConfiguration
	if err := mapstructure.Decode(a, &cfgA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cfgB); err != nil {
		return false, err
	}

	if len(cfgA.Events) != len(cfgB.Events) {
		return false, nil
	}
	seen := make(map[string]bool)
	for _, e := range cfgA.Events {
		seen[e] = true
	}
	for _, e := range cfgB.Events {
		if !seen[e] {
			return false, nil
		}
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
	for _, e := range reqCfg.Events {
		if seen[e] {
			continue
		}
		seen[e] = true
		mergedEvents = append(mergedEvents, e)
	}

	merged := WebhookConfiguration{Events: mergedEvents}
	changed, err := h.CompareConfig(curCfg, merged)
	if err != nil {
		return nil, false, err
	}
	return merged, !changed, nil
}

func (h *SentryWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
