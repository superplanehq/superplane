package incident

import (
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config stored with the webhook (events list).
// incident.io does not expose an API to create endpoints; the user adds the URL in the dashboard
// and pastes the signing secret into the trigger configuration.
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

	for _, eventB := range configB.Events {
		if !slices.Contains(configA.Events, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *IncidentIOWebhookHandler) Merge(current, requested any) (any, bool, error) {
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
