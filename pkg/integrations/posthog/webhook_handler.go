package posthog

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// WebhookTokenHeader carries the shared secret PostHog sends back to SuperPlane.
//
// PostHog's webhook destination does not sign its payloads, so there is no
// signature to verify. Instead SuperPlane generates a secret per webhook, stores
// it encrypted, and sets it as a header on the destination it creates. Requests
// that do not present the secret are rejected.
const WebhookTokenHeader = "X-SuperPlane-Token"

// webhookSecretSize is the byte length of the generated shared secret.
const webhookSecretSize = 32

// WebhookConfiguration is the config stored with the webhook. Webhooks are
// shared between triggers whose configuration matches, so everything that
// changes the destination in PostHog belongs here.
type WebhookConfiguration struct {
	ProjectID          string   `json:"projectId" mapstructure:"projectId"`
	Events             []string `json:"events" mapstructure:"events"`
	FilterTestAccounts bool     `json:"filterTestAccounts" mapstructure:"filterTestAccounts"`
}

// WebhookMetadata is stored after Setup. It holds the hog function ID and the
// project it lives in, both needed to delete the destination later.
type WebhookMetadata struct {
	ProjectID     string `json:"projectId" mapstructure:"projectId"`
	HogFunctionID string `json:"hogFunctionId" mapstructure:"hogFunctionId"`
}

type PostHogWebhookHandler struct{}

func (h *PostHogWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	if configA.ProjectID != configB.ProjectID || configA.FilterTestAccounts != configB.FilterTestAccounts {
		return false, nil
	}

	//
	// The event list drives the filter on the PostHog side, so two triggers can
	// only share a destination when they ask for the same events. Order is a UI
	// artifact and must not split an otherwise reusable webhook.
	//
	eventsA := slices.Clone(configA.Events)
	eventsB := slices.Clone(configB.Events)
	slices.Sort(eventsA)
	slices.Sort(eventsB)

	return slices.Equal(eventsA, eventsB), nil
}

func (h *PostHogWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// Setup creates a webhook destination in PostHog pointing back at SuperPlane.
// The shared secret is generated and stored first, so a delivery that arrives
// before Setup returns can still be authenticated.
func (h *PostHogWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostHog client: %w", err)
	}

	secret, err := crypto.Base64String(webhookSecretSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate webhook secret: %w", err)
	}

	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook secret: %w", err)
	}

	events := make([]HogFunctionEventFilter, 0, len(config.Events))
	for _, event := range config.Events {
		events = append(events, HogFunctionEventFilter{ID: event, Type: "events"})
	}

	hogFunction, err := client.CreateHogFunction(config.ProjectID, CreateHogFunctionRequest{
		Type:        HogFunctionTypeDestination,
		TemplateID:  WebhookTemplateID,
		Name:        "SuperPlane",
		Description: "Sends matching events to a SuperPlane workflow. Managed by SuperPlane.",
		Enabled:     true,
		Inputs: map[string]HogFunctionInput{
			"url":    {Value: ctx.Webhook.GetURL()},
			"method": {Value: http.MethodPost},
			"body":   {Value: webhookBody()},
			"headers": {Value: map[string]string{
				"Content-Type":     "application/json",
				WebhookTokenHeader: secret,
			}},
		},
		Filters: HogFunctionFilters{
			Events:             events,
			FilterTestAccounts: config.FilterTestAccounts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook destination in PostHog: %w", err)
	}

	return WebhookMetadata{
		ProjectID:     config.ProjectID,
		HogFunctionID: hogFunction.ID,
	}, nil
}

// Cleanup deletes the webhook destination from PostHog when the trigger is removed.
func (h *PostHogWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.HogFunctionID == "" || metadata.ProjectID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create PostHog client: %w", err)
	}

	if err := client.DeleteHogFunction(metadata.ProjectID, metadata.HogFunctionID); err != nil {
		//
		// If the destination is already gone in PostHog, there is nothing to
		// clean up and the record can be dropped.
		//
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("failed to delete webhook destination from PostHog: %w", err)
	}

	return nil
}

// webhookBody is the templated payload PostHog renders per event. The braces are
// Hog template placeholders that PostHog expands before delivery.
func webhookBody() map[string]any {
	return map[string]any{
		"event":  "{event}",
		"person": "{person}",
		"project": map[string]any{
			"id":   "{project.id}",
			"name": "{project.name}",
			"url":  "{project.url}",
		},
	}
}
