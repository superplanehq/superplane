package incident

import (
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config stored with the webhook (events only).
// The signing secret is stored in the encrypted webhook.Secret via the setSecret action.
type WebhookConfiguration struct {
	Events []string `json:"events"`
}

// WebhookMetadata is stored after Setup. We do not create an endpoint via API, so metadata can be empty.
type WebhookMetadata struct{}

type IncidentIOWebhookHandler struct{}

func (h *IncidentIOWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	// Reuse same webhook when requested events are subset of existing (user narrowed) or
	// existing events are subset of requested (user expanded). That keeps the URL stable.
	eventsMatch := subsetOrEqual(configA.Events, configB.Events) || subsetOrEqual(configB.Events, configA.Events)
	return eventsMatch, nil
}

// subsetOrEqual returns true when a is subset of or equal to b (every element of a is in b).
func subsetOrEqual(a, b []string) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}

func (h *IncidentIOWebhookHandler) Merge(current, requested any) (any, bool, error) {
	cur := WebhookConfiguration{}
	req := WebhookConfiguration{}

	if err := mapstructure.Decode(current, &cur); err != nil {
		return nil, false, err
	}
	if err := mapstructure.Decode(requested, &req); err != nil {
		return nil, false, err
	}

	changed := false

	// Merge events when requested is superset of current (user added more events) so we update in place and keep same URL.
	if subsetOrEqual(cur.Events, req.Events) && !slices.Equal(cur.Events, req.Events) {
		cur.Events = req.Events
		changed = true
	}

	if changed {
		return cur, true, nil
	}
	return current, false, nil
}

// Setup does not call incident.io (no API to create webhook endpoints).
// The user adds the webhook URL in incident.io Settings > Webhooks and pastes the signing secret into the trigger.
func (h *IncidentIOWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	_ = ctx
	return WebhookMetadata{}, nil
}

func (h *IncidentIOWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	_ = ctx
	return nil
}
