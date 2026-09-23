package productive

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// remoteWebhookEvents is the Productive.io event_id pair the onTask trigger
// can listen for. Productive.io webhooks are organization-wide and fire for
// one event each, so Setup registers both against the same SuperPlane URL.
// CompareConfig stays project-scoped, so every node watching a project shares
// one SuperPlane webhook and HandleWebhook drops other projects and actions.
var remoteWebhookEvents = []struct {
	eventID   int
	eventName string
}{
	{EventNewTask, TaskCreatedEvent},
	{EventUpdatedTask, TaskUpdatedEvent},
}

// WebhookConfiguration scopes a SuperPlane webhook to one Productive.io
// project. Remote webhooks are organization-wide; the project id is used to
// share one SuperPlane webhook among nodes and to filter deliveries.
type WebhookConfiguration struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
}

// WebhookMetadata is stored on the webhook record after remote webhooks are
// created, so Cleanup can delete each Productive.io webhook again.
type WebhookMetadata struct {
	IDs []string `json:"ids" mapstructure:"ids"`
	// ID is the single remote id the invented API stored. Cleanup still
	// deletes it so an old record does not leak a Productive.io webhook.
	ID string `json:"id,omitempty" mapstructure:"id,omitempty"`
}

func (m WebhookMetadata) remoteIDs() []string {
	ids := make([]string, 0, len(m.IDs)+1)
	seen := map[string]bool{}
	for _, id := range append(m.IDs, m.ID) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// ProductiveWebhookHandler creates and tears down the Productive.io webhooks
// the onTask trigger subscribes to.
type ProductiveWebhookHandler struct{}

func (h *ProductiveWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *ProductiveWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.ProjectID == configB.ProjectID, nil
}

func (h *ProductiveWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook config: %v", err)
	}
	if strings.TrimSpace(config.ProjectID) == "" {
		return nil, fmt.Errorf("project is required")
	}

	ids := make([]string, 0, len(remoteWebhookEvents))
	tokens := make([]string, 0, len(remoteWebhookEvents))

	for _, event := range remoteWebhookEvents {
		webhook, err := client.CreateWebhook(ctx.Webhook.GetURL(), event.eventID, event.eventName)
		if err != nil {
			h.deleteRemoteWebhooks(client, ids)

			if errors.Is(err, ErrWebhooksLimitExceeded) {
				return nil, fmt.Errorf("Productive.io does not offer webhooks on this plan, so the On Task trigger cannot be set up: %w", err)
			}

			return nil, fmt.Errorf("error creating webhook: %v", err)
		}

		ids = append(ids, webhook.ID)
		if token := strings.TrimSpace(webhook.SignatureToken); token != "" && !slices.Contains(tokens, token) {
			tokens = append(tokens, token)
		}
	}

	if len(tokens) == 0 {
		h.deleteRemoteWebhooks(client, ids)
		return nil, fmt.Errorf("productive.io did not return a webhook signature token")
	}

	if err := ctx.Webhook.SetSecret([]byte(strings.Join(tokens, "\n"))); err != nil {
		h.deleteRemoteWebhooks(client, ids)
		return nil, fmt.Errorf("error storing webhook signature token: %v", err)
	}

	return &WebhookMetadata{IDs: ids}, nil
}

func (h *ProductiveWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %v", err)
	}

	ids := metadata.remoteIDs()
	if len(ids) == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	return h.deleteRemoteWebhooks(client, ids)
}

func (h *ProductiveWebhookHandler) deleteRemoteWebhooks(client *Client, ids []string) error {
	var first error
	for _, id := range ids {
		if err := client.DeleteWebhook(id); err != nil && first == nil {
			first = fmt.Errorf("error deleting webhook: %v", err)
		}
	}
	return first
}
