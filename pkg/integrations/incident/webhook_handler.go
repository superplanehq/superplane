package incident

import (
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config stored with the webhook (events and signing secret).
// incident.io does not expose an API to create endpoints; the user adds the URL in the dashboard
// and pastes the signing secret into the trigger configuration.
// SigningSecret is included so webhook reuse is keyed on (events, signingSecret); triggers with
// different secrets must not share a webhook or one node's 403 can block others.
type WebhookConfiguration struct {
	Events        []string `json:"events"`
	SigningSecret string   `json:"signingSecret"`
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

	// Allow reuse when secrets match, or when the existing webhook has no secret yet
	// (user added the trigger first, then pasted the signing secret; we merge it in so the URL stays the same).
	if configA.SigningSecret != configB.SigningSecret && configA.SigningSecret != "" {
		return false, nil
	}

	return true, nil
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

	// Merge signing secret only when the existing webhook had none (user added it after creating the trigger).
	if cur.SigningSecret == "" && req.SigningSecret != "" {
		cur.SigningSecret = req.SigningSecret
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
