package launchdarkly

import (
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config stored with the webhook.
// The signing secret is stored in the encrypted webhook.Secret via the setSecret action.
type WebhookConfiguration struct {
	Events []string `json:"events"`
}

// WebhookMetadata is stored after Setup. We do not create a webhook via API,
// so metadata can be empty.
type WebhookMetadata struct{}

type LaunchDarklyWebhookHandler struct{}

func (h *LaunchDarklyWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	eventsMatch := subsetOrEqual(configA.Events, configB.Events) || subsetOrEqual(configB.Events, configA.Events)
	return eventsMatch, nil
}

// subsetOrEqual returns true when a is a subset of or equal to b.
func subsetOrEqual(a, b []string) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}

func (h *LaunchDarklyWebhookHandler) Merge(current, requested any) (any, bool, error) {
	cur := WebhookConfiguration{}
	req := WebhookConfiguration{}

	if err := mapstructure.Decode(current, &cur); err != nil {
		return nil, false, err
	}
	if err := mapstructure.Decode(requested, &req); err != nil {
		return nil, false, err
	}

	changed := false

	if subsetOrEqual(cur.Events, req.Events) && !slices.Equal(cur.Events, req.Events) {
		cur.Events = req.Events
		changed = true
	}

	if changed {
		return cur, true, nil
	}
	return current, false, nil
}

// Setup does not call LaunchDarkly API (no API to create webhook endpoints programmatically with signing).
// The user adds the webhook URL in LaunchDarkly Integrations > Webhooks and pastes the signing secret into the trigger.
func (h *LaunchDarklyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return WebhookMetadata{}, nil
}

func (h *LaunchDarklyWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}
